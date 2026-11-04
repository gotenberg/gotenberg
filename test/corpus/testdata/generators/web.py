import os, zipfile
from PIL import Image, ImageDraw, ImageFont
C=os.environ["C"]+"/web"
def w(p,s):
    os.makedirs(os.path.dirname(f"{C}/{p}"),exist_ok=True); open(f"{C}/{p}","w",encoding="utf-8").write(s)
def font(sz):
    for f in ["/System/Library/Fonts/Supplemental/Arial Bold.ttf","/System/Library/Fonts/Supplemental/Arial.ttf"]:
        try: return ImageFont.truetype(f,sz)
        except Exception: pass
    return ImageFont.load_default()
# logo
im=Image.new("RGB",(240,80),(20,50,110)); d=ImageDraw.Draw(im); d.text((120,40),"NORTHWIND",font=font(30),fill="white",anchor="mm"); im.save(f"{C}/invoice/logo.png")
rows=[("Consulting",10,150),("Hosting",1,400),("Software license",5,120),("Support hours",4,100),("Training session",2,200),("Domain registration",1,50),("SSL certificate",1,40),("Backup storage",2,30),("Monitoring",1,100),("Security audit",1,150),("Setup fee",1,100),("Miscellaneous",1,50)]
sub=sum(q*u for _,q,u in rows); assert sub==3850 and sub*1.2==4620
tr="".join(f"<tr><td>{i+1}</td><td>{n}</td><td class='r'>{q}</td><td class='r'>{u:,.2f}</td><td class='r'>{q*u:,.2f}</td></tr>\n" for i,(n,q,u) in enumerate(rows))
w("invoice/style.css","""@page { size: A4; margin: 15mm }
body{font-family:Helvetica,Arial,sans-serif;font-size:11pt;color:#222}
h1{color:#143272}h2{border-bottom:1px solid #143272;padding-bottom:2px}
table{border-collapse:collapse;width:100%}
th{background-color:#143272;color:#fff;text-align:left;padding:6px}
td{padding:6px;border-bottom:1px solid #ddd}.r{text-align:right}
.total{font-weight:bold;font-size:14pt}
h2.terms{break-before:page}
""")
w("invoice/index.html",f"""<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Invoice INV-2026-0042</title><link rel="stylesheet" href="style.css"></head>
<body>
<img src="logo.png" alt="Northwind logo" width="240" height="80">
<h1>Invoice INV-2026-0042</h1>
<h3>Northwind Traders</h3><p>1 Harbour Road, Seattle, WA 98101</p>
<h3>Bill to: Contoso Ltd</h3><p>22 Market Street, London EC2N 1AB</p>
<h2>Line items</h2>
<table><thead><tr><th>#</th><th>Description</th><th>Qty</th><th>Unit price</th><th>Amount</th></tr></thead>
<tbody>
{tr}</tbody></table>
<h2>Totals</h2>
<table><tr><td>Subtotal</td><td class="r">3,850.00 EUR</td></tr><tr><td>VAT 20%</td><td class="r">770.00 EUR</td></tr><tr class="total"><td>Total</td><td class="r">4,620.00 EUR</td></tr></table>
<h2 class="terms">Payment terms</h2>
<p>Payment is due within 30 days of the invoice date. Late payments incur 1.5% monthly interest.</p>
<h3>Bank details</h3><p>IBAN FR76 3000 6000 0112 3456 7890 189, BIC AGRIFRPP</p>
</body></html>""")
w("invoice/header.html","""<html><head><style>body{font-size:9px;margin:0 15mm;width:100%;color:#555}</style></head>
<body><span class="title"></span> <span style="float:right" class="date"></span></body></html>""")
w("invoice/footer.html","""<html><head><style>body{font-size:9px;margin:0 15mm;width:100%;text-align:center;color:#555}</style></head>
<body>Page <span class="pageNumber"></span> of <span class="totalPages"></span></body></html>""")
w("wait/index.html","""<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Wait</title></head>
<body><div id="chart">Loading…</div>
<script>setTimeout(function(){var c=document.getElementById('chart');c.textContent='Revenue: 42,000 EUR';c.classList.add('ready');window.chartReady=true;window.status='ready';},1000);</script></body></html>""")
w("console-error/index.html","""<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Console error</title></head>
<body><p>Console error page</p><script>throw new Error("boom");</script></body></html>""")
# markdown
sh=Image.new("RGB",(640,360),(240,244,250)); d=ImageDraw.Draw(sh); d.rectangle((20,20,620,60),fill=(20,50,110)); d.text((320,200),"SCREENSHOT",font=font(40),fill=(20,50,110),anchor="mm"); sh.save(f"{C}/markdown/screenshot.png")
w("markdown/style.css","body{font-family:Helvetica,Arial,sans-serif}table,th,td{border:1px solid #888;border-collapse:collapse;padding:4px}pre{background:#f4f4f4;padding:8px}img{max-width:100%}")
w("markdown/index.html","""<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Release notes 9.0.0</title><link rel="stylesheet" href="style.css"></head>
<body>
{{ toHTML "release-notes.md" }}
<hr>
{{ toHTML "changelog.md" }}
</body></html>""")
w("markdown/release-notes.md","""# Release notes 9.0.0

Welcome to the 9.0.0 release.

## Highlights

| Feature | Status |
|---------|--------|
| Chromium routes | stable |
| PDF remediation | new |

## Code sample

```go
func main() {
    fmt.Println("CODE-BLOCK-MARKER")
}
```

## Checklist

~~Removed legacy flag~~

- [x] Done item DONE-TASK
- [ ] Open item OPEN-TASK

## Glossary

Gotenberg
:   A document-to-PDF API.

LibreOffice
:   Office suite used for conversions.

## Links

Visit https://gotenberg.dev for docs.

![Screenshot](screenshot.png)

<script>document.title='pwned'</script>SANITIZE-ME-MARKER
""")
w("markdown/changelog.md","""# Changelog

- 9.0.0 - CHANGELOG-V9 new routes
- 8.25.0 - CHANGELOG-V8 maintenance
- 8.0.0 - first 8.x release
""")
# landing
secs="".join(f"<section class='block'><h2>Feature {i}</h2><div class='box'>Box {i} painted content</div><p>{'Lorem ipsum dolor sit amet. '*20}</p></section>" for i in range(1,9))
w("landing/index.html",f"""<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Landing</title>
<style>
body{{background:transparent;margin:0;font-family:Arial,sans-serif}}
nav{{padding:12px;background:#222;color:#fff}}@media print{{nav{{display:none}}}}
.hero{{height:600px;background:linear-gradient(135deg,#6a11cb,#2575fc);color:#fff;text-align:center;padding-top:200px;box-sizing:border-box}}
#cta{{display:inline-block;padding:14px 28px;background:#ffb703;color:#000;border:0;font-size:20px}}
.block{{min-height:320px;padding:20px}}.box{{background:#cde;padding:20px;width:300px}}
.wide{{display:none}}@media (min-width:1200px){{.wide{{display:block;background:#fed;padding:20px}}}}
</style></head><body>
<nav>Home | Pricing | Contact</nav>
<div class="hero"><h1>Ship documents faster</h1><button id="cta">Start free trial</button></div>
<div class="wide">WIDE-LAYOUT-MARKER</div>
{secs}
</body></html>""")
# i18n
L=[("AR","مرحبا بالعالم، هذه فقرة عربية للاختبار","rtl"),("HE","שלום עולם, זו פסקה בעברית לבדיקה","rtl"),("ZH","你好，世界。这是一段简体中文测试文字。",""),("JA","こんにちは世界。これは日本語のテスト段落です。",""),("KO","안녕하세요 세계. 이것은 한국어 테스트 단락입니다.",""),("TH","สวัสดีชาวโลก นี่คือย่อหน้าภาษาไทยสำหรับทดสอบ",""),("HI","नमस्ते दुनिया, यह परीक्षण के लिए हिंदी अनुच्छेद है।",""),("EL","Γειά σου κόσμε, αυτή είναι ελληνική παράγραφος.",""),("RU","Привет, мир! Это русский абзац для проверки.",""),("EMOJI","😀 🚀 🎉 ❤️ 🌍",""),("MATH","∑ ∫ ≠ ∞ √ π ≤ ≥","")]
import json; json.dump(L,open(os.environ["B"]+"/i18n.json","w"),ensure_ascii=False)
ps="\n".join(f'<p{" dir=%s"%chr(34)+d+chr(34) if d else ""}>{m}: {t}</p>' for m,t,d in L)
w("i18n/index.html",f'<!doctype html><html lang="en"><head><meta charset="utf-8"><title>i18n</title></head><body><h1>Multilingual</h1>\n{ps}\n</body></html>')
# print-css
chap=lambda n,m:f"<h2 id='c{n}'>Chapter {n}</h2><p>{m}</p><p>{'Handbook body text. '*40}</p>"
w("print-css/print.css","@page { size: A5; margin: 15mm }\nh2 { break-before: page }\n@media print { nav { display: none } }\nbody{font-family:Georgia,serif}\n")
w("print-css/index.html",f"""<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Handbook</title><link rel="stylesheet" href="print.css"></head>
<body><nav><a href="#c1">One</a> <a href="#c2">Two</a> <a href="#c3">Three</a> NAV-MARKER</nav><h1>Handbook</h1><p>Intro paragraph.</p>
{chap(1,"CHAPTER-ONE")}{chap(2,"CHAPTER-TWO")}{chap(3,"CHAPTER-THREE")}</body></html>""")
# wide table
th="".join(f"<th>C{i:02d}</th>" for i in range(1,31))
tb="".join("<tr>"+"".join(f"<td>r{r}c{c}</td>" for c in range(1,31))+"</tr>" for r in range(1,41))
w("wide-table/index.html",f'<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Wide table</title><style>table{{border-collapse:collapse}}td,th{{border:1px solid #999;padding:2px 6px;white-space:nowrap}}</style></head><body><table><thead><tr>{th}</tr></thead><tbody>{tb}</tbody></table></body></html>')
with zipfile.ZipFile(f"{C}/site.zip","w",zipfile.ZIP_DEFLATED) as z:
    for f in ["index.html","style.css","logo.png"]: z.write(f"{C}/invoice/{f}",f)
