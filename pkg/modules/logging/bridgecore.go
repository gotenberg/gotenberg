package logging

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

type sharedBridgeState struct {
	mu     sync.RWMutex
	target zapcore.Core
	buffer []bufferedLog
}

type bridgeCore struct {
	shared *sharedBridgeState
	fields []zapcore.Field
	level  zapcore.LevelEnabler
}

type bufferedLog struct {
	ent    zapcore.Entry
	fields []zapcore.Field
}

// If specified in the fields, the bridge core will ignore the corresponding
// entries.
const internalKey = "internal"

func newBridgeCore(level zapcore.LevelEnabler) *bridgeCore {
	return &bridgeCore{
		shared: &sharedBridgeState{
			buffer: make([]bufferedLog, 0, 1000),
		},
		level: level,
	}
}

func (b *bridgeCore) SetTarget(core zapcore.Core) {
	b.shared.mu.Lock()
	defer b.shared.mu.Unlock()

	b.shared.target = core

	for _, log := range b.shared.buffer {
		if ce := b.shared.target.Check(log.ent, nil); ce != nil {
			ce.Write(log.fields...)
		}
	}

	b.shared.buffer = nil
}

func (b *bridgeCore) Enabled(lvl zapcore.Level) bool {
	b.shared.mu.RLock()
	if b.shared.target != nil {
		defer b.shared.mu.RUnlock()
		return b.shared.target.Enabled(lvl)
	}
	b.shared.mu.RUnlock()

	return b.level.Enabled(lvl)
}

func (b *bridgeCore) With(fields []zapcore.Field) zapcore.Core {
	return &bridgeCore{
		shared: b.shared,
		level:  b.level,
		fields: append(b.fields[:len(b.fields):len(b.fields)], fields...),
	}
}

func (b *bridgeCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	b.shared.mu.RLock()
	defer b.shared.mu.RUnlock()

	if b.shared.target != nil {
		return b.shared.target.Check(ent, ce)
	}

	return ce.AddCore(ent, b)
}

func (b *bridgeCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	allFields := make([]zapcore.Field, 0, len(b.fields)+len(fields))
	allFields = append(allFields, b.fields...)
	allFields = append(allFields, fields...)

	for _, f := range allFields {
		if f.Key == internalKey {
			return nil
		}
	}

	b.shared.mu.Lock()
	defer b.shared.mu.Unlock()

	if b.shared.target != nil {
		return b.shared.target.Write(ent, allFields)
	}

	if len(b.shared.buffer) < 1000 {
		b.shared.buffer = append(b.shared.buffer, bufferedLog{
			ent:    ent,
			fields: allFields,
		})
	}

	return nil
}

func (b *bridgeCore) Sync() error {
	b.shared.mu.RLock()
	defer b.shared.mu.RUnlock()

	if b.shared.target != nil {
		return b.shared.target.Sync()
	}

	return nil
}

// Interface guard.
var (
	_ zapcore.Core = (*bridgeCore)(nil)
)
