package iterm2

import (
	"fmt"
	"strconv"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

// Window represents an iTerm2 Window
type Window interface {
	SetTitle(s string) error
	CreateTab() (Tab, error)
	ListTabs() ([]Tab, error)
	Activate() error
	Frame() (Frame, error)
	SetFrame(f Frame) error
}

// Frame is a window's position and size in screen points. The origin is the
// bottom left corner of the main display.
type Frame struct {
	X, Y, Width, Height int32
}

type window struct {
	c       *client.Client
	id      string
	session string
}

func (w *window) CreateTab() (Tab, error) {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{
				WindowId: str(w.id),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create tab for window %q: %w", w.id, err)
	}
	ctr := resp.GetCreateTabResponse()
	if ctr.GetStatus() != api.CreateTabResponse_OK {
		return nil, fmt.Errorf("unexpected tab status: %s", ctr.GetStatus())
	}
	return &tab{
		c:        w.c,
		id:       strconv.Itoa(int(ctr.GetTabId())),
		windowID: w.id,
	}, nil
}

func (w *window) ListTabs() ([]Tab, error) {
	list := []Tab{}
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, window := range resp.GetListSessionsResponse().GetWindows() {
		if window.GetWindowId() != w.id {
			continue
		}
		for _, t := range window.GetTabs() {
			list = append(list, &tab{
				c:        w.c,
				id:       t.GetTabId(),
				windowID: w.id,
			})
		}
	}
	return list, nil
}

func (w *window) SetTitle(s string) error {
	_, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: str(fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)),
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &w.id,
					},
				},
			},
		},
	})
	return err
}

func (w *window) Activate() error {
	_, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{ActivateRequest: &api.ActivateRequest{
			Identifier:       &api.ActivateRequest_WindowId{WindowId: w.id},
			OrderWindowFront: b(true),
		}},
	})
	return err
}

// Frame returns where the window currently sits on screen.
func (w *window) Frame() (Frame, error) {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return Frame{}, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, window := range resp.GetListSessionsResponse().GetWindows() {
		if window.GetWindowId() != w.id {
			continue
		}
		frame := window.GetFrame()
		return Frame{
			X:      frame.GetOrigin().GetX(),
			Y:      frame.GetOrigin().GetY(),
			Width:  frame.GetSize().GetWidth(),
			Height: frame.GetSize().GetHeight(),
		}, nil
	}
	return Frame{}, fmt.Errorf("window %q not found", w.id)
}

// SetFrame moves and resizes the window. iTerm2 clamps the height to the
// display the window lands on, but does not clamp the width.
func (w *window) SetFrame(f Frame) error {
	value := fmt.Sprintf(`{"origin":{"x":%d,"y":%d},"size":{"width":%d,"height":%d}}`,
		f.X, f.Y, f.Width, f.Height)
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SetPropertyRequest{
			SetPropertyRequest: &api.SetPropertyRequest{
				Identifier: &api.SetPropertyRequest_WindowId{WindowId: w.id},
				Name:       str("frame"),
				JsonValue:  &value,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not set frame of window %q: %w", w.id, err)
	}
	if status := resp.GetSetPropertyResponse().GetStatus(); status != api.SetPropertyResponse_OK {
		return fmt.Errorf("unexpected status setting frame of window %q: %s", w.id, status)
	}
	return nil
}
