package scenario

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"sync/atomic"
	"time"
)

// The timestamp authority below is a trimmed port of the one litho tests its
// signing against (internal/pkitest), which Go's internal rule keeps out of
// reach. It grants every request: litho checks a token's signature and
// imprint, not that its chain leads to a trusted root.

// OIDs the timestamp authority writes.
var (
	oidSHA256          = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidECDSAWithSHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidSignedData      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidTSTInfo         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	oidContentType     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidMessageDigest   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidSigningCertV2   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47}
	oidTSAPolicy       = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1}
)

// maxTimestampRequestBytes bounds a timestamp request body. Gotenberg sends a
// few hundred bytes.
const maxTimestampRequestBytes = 1 << 16

// messageImprint is the MessageImprint of RFC 3161, 2.4.1.
type messageImprint struct {
	HashAlgorithm pkix.AlgorithmIdentifier
	HashedMessage []byte
}

// timeStampReq is the TimeStampReq of RFC 3161, 2.4.1.
type timeStampReq struct {
	Version        int
	MessageImprint messageImprint
	ReqPolicy      asn1.ObjectIdentifier `asn1:"optional"`
	Nonce          *big.Int              `asn1:"optional"`
	CertReq        bool                  `asn1:"optional"`
}

// tstInfo is the TSTInfo of RFC 3161, 2.4.2, without the optional fields the
// authority does not write.
type tstInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time `asn1:"generalized"`
	Nonce          *big.Int  `asn1:"optional"`
}

// timestampAuthority answers RFC 3161 requests with tokens its own
// certificate signs.
type timestampAuthority struct {
	cert   *x509.Certificate
	key    *ecdsa.PrivateKey
	serial atomic.Int64
}

// newTimestampAuthority issues a root and the timestamp authority's
// certificate under it.
func newTimestampAuthority() (*timestampAuthority, error) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate root key: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate timestamp authority key: %w", err)
	}

	notBefore := time.Now().AddDate(-1, 0, 0).UTC().Truncate(time.Second)
	notAfter := notBefore.AddDate(11, 0, 0)

	root := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "integration root"},
		NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage: x509.KeyUsageCertSign, BasicConstraintsValid: true, IsCA: true,
	}

	// RFC 3161, 2.3 makes the timeStamping extended key usage critical, which
	// crypto/x509 leaves it not.
	eku, err := asn1.Marshal([]asn1.ObjectIdentifier{{1, 3, 6, 1, 5, 5, 7, 3, 8}})
	if err != nil {
		return nil, fmt.Errorf("encode extended key usage: %w", err)
	}

	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "integration timestamp authority"},
		NotBefore: notBefore, NotAfter: notAfter,
		KeyUsage:        x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{{Id: asn1.ObjectIdentifier{2, 5, 29, 37}, Critical: true, Value: eku}},
	}, root, key.Public(), rootKey)
	if err != nil {
		return nil, fmt.Errorf("issue the timestamp authority certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse the timestamp authority certificate: %w", err)
	}

	return &timestampAuthority{cert: cert, key: key}, nil
}

// ServeHTTP answers an RFC 3161 request over HTTP (RFC 3161, 3.4) with a
// granted token.
func (a *timestampAuthority) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxTimestampRequestBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req timeStampReq
	_, err = asn1.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tst, err := asn1.Marshal(tstInfo{
		Version: 1, Policy: oidTSAPolicy, MessageImprint: req.MessageImprint,
		SerialNumber: big.NewInt(a.serial.Add(1)),
		GenTime:      time.Now().UTC().Truncate(time.Second), Nonce: req.Nonce,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := a.signedData(tst, req.CertReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// PKIStatusInfo { granted (0) } (RFC 3161, 2.4.2).
	status, err := asn1.Marshal(struct{ Status int }{0})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := wrap(asn1.ClassUniversal, asn1.TagSequence, slices.Concat(status, token))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/timestamp-reply")
	_, _ = w.Write(resp)
}

// signedData returns a ContentInfo holding a SignedData that encapsulates the
// TSTInfo content, signed under signed content-type, message-digest and ESS
// signing-certificate-v2 attributes (RFC 3161, 2.4.2; RFC 5816), with the
// authority's certificate when withCert is set.
func (a *timestampAuthority) signedData(content []byte, withCert bool) ([]byte, error) {
	var firstErr error
	der := func(v any) []byte {
		b, err := asn1.Marshal(v)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return b
	}
	seq := func(content ...[]byte) []byte {
		b, err := wrap(asn1.ClassUniversal, asn1.TagSequence, slices.Concat(content...))
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return b
	}
	tagged := func(class, tag int, content []byte) []byte {
		b, err := wrap(class, tag, content)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return b
	}

	digest := sha256.Sum256(content)
	certHash := sha256.Sum256(a.cert.Raw)
	issuerSerial := seq(seq(tagged(asn1.ClassContextSpecific, 4, a.cert.RawIssuer)), der(a.cert.SerialNumber))
	essCert := seq(seq(seq(der(certHash[:]), issuerSerial)))
	attr := func(oid asn1.ObjectIdentifier, value []byte) []byte {
		return seq(der(oid), tagged(asn1.ClassUniversal, asn1.TagSet, value))
	}

	attrs := [][]byte{
		attr(oidContentType, der(oidTSTInfo)),
		attr(oidMessageDigest, der(digest[:])),
		attr(oidSigningCertV2, essCert),
	}
	// DER orders the elements of a SET OF.
	slices.SortFunc(attrs, bytes.Compare)
	attrSet := tagged(asn1.ClassUniversal, asn1.TagSet, slices.Concat(attrs...))
	if firstErr != nil {
		return nil, firstErr
	}

	h := sha256.Sum256(attrSet)
	sig, err := a.key.Sign(rand.Reader, h[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("sign the token: %w", err)
	}

	// The signed attributes go in as [0] IMPLICIT, constructed.
	signedAttrs := slices.Clone(attrSet)
	signedAttrs[0] = 0xA0

	sha256Alg := der(pkix.AlgorithmIdentifier{Algorithm: oidSHA256})
	sigAlg := der(pkix.AlgorithmIdentifier{Algorithm: oidECDSAWithSHA256})
	sid := seq(a.cert.RawIssuer, der(a.cert.SerialNumber))
	signerInfo := seq(der(1), sid, sha256Alg, signedAttrs, sigAlg, der(sig))

	encap := seq(der(oidTSTInfo), tagged(asn1.ClassContextSpecific, 0, der(content)))
	body := slices.Concat(der(3), tagged(asn1.ClassUniversal, asn1.TagSet, sha256Alg), encap)
	if withCert {
		body = append(body, tagged(asn1.ClassContextSpecific, 0, a.cert.Raw)...)
	}
	body = append(body, tagged(asn1.ClassUniversal, asn1.TagSet, signerInfo)...)

	token := seq(der(oidSignedData), tagged(asn1.ClassContextSpecific, 0, seq(body)))
	if firstErr != nil {
		return nil, firstErr
	}

	return token, nil
}

// wrap returns the DER of a constructed element of class and tag holding
// content.
func wrap(class, tag int, content []byte) ([]byte, error) {
	return asn1.Marshal(asn1.RawValue{Class: class, Tag: tag, IsCompound: true, Bytes: content})
}
