package missionweaveprotocol

import (
	"errors"
	"fmt"
)

const frameSchemaName = "websocket-frame.schema.json"

// FrameCodec canonicalizes and schema-validates generic MissionWeaveProtocol WebSocket frames.
type FrameCodec struct {
	catalog *SchemaCatalog
}

// NewFrameCodec constructs a codec over the protocol schemas embedded in this SDK build.
func NewFrameCodec() (*FrameCodec, error) {
	catalog, err := NewEmbeddedSchemaCatalog()
	if err != nil {
		return nil, err
	}
	return &FrameCodec{catalog: catalog}, nil
}

// DecodeFrame strictly parses and validates one UTF-8 JSON frame.
func (codec *FrameCodec) DecodeFrame(document []byte) (map[string]any, error) {
	if codec == nil || codec.catalog == nil {
		return nil, errors.New("FrameCodec is not initialized")
	}
	value, err := DecodeJSON(document)
	if err != nil {
		return nil, err
	}
	frame, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("WebSocket frame must be a JSON object")
	}
	if err := codec.catalog.validateValue(frameSchemaName, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

// EncodeFrame validates one generic frame and returns canonical RFC 8785 JSON.
func (codec *FrameCodec) EncodeFrame(frame map[string]any) ([]byte, error) {
	if codec == nil || codec.catalog == nil {
		return nil, errors.New("FrameCodec is not initialized")
	}
	if frame == nil {
		return nil, errors.New("WebSocket frame must not be nil")
	}
	if err := codec.catalog.validateValue(frameSchemaName, frame); err != nil {
		return nil, err
	}
	encoded, err := MarshalCanonicalJSON(frame)
	if err != nil {
		return nil, fmt.Errorf("encode WebSocket frame: %w", err)
	}
	return encoded, nil
}
