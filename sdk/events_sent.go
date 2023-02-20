package sdk

import (
	"context"

	"nhooyr.io/websocket/wsjson"
)

type SetFeedbackEvent struct {
	Event   string            `json:"event,omitempty"`
	Context string            `json:"context,omitempty"`
	Payload map[string]string `json:"payload,omitempty"`
}

func (p *Plugin) SetFeedback(c string, payload map[string]string) error {
	e := SetFeedbackEvent{
		Event:   "setFeedback",
		Context: c,
		Payload: payload,
	}

	ctx := context.Background()
	if err := wsjson.Write(ctx, p.conn, e); err != nil {
		return err
	}

	return nil
}
