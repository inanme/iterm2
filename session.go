package iterm2

import (
	"fmt"
	"os"

	"marwan.io/iterm2/api"
	"marwan.io/iterm2/client"
)

// Session represents an iTerm2 Session which is a pane
// within a Tab where the terminal is active
type Session interface {
	SendText(s string) error
	Activate(selectTab, orderWindowFront bool) error
	SplitPane(opts SplitPaneOptions) (Session, error)
	GetSessionID() string
}

// SplitPaneOptions for customizing the new pane session.
// More options can be added here as needed
type SplitPaneOptions struct {
	Vertical bool
	// Profile is the name of the iTerm2 profile the new pane opens with.
	// iTerm2 rejects a split that names no profile -- it answers
	// MALFORMED_CUSTOM_PROFILE_PROPERTY even when no custom properties are
	// sent at all -- so when this is empty DefaultProfile is used.
	Profile string
}

// DefaultProfile returns the profile SplitPane uses when SplitPaneOptions
// names none: the profile of the session iTerm2 exported ITERM_PROFILE for,
// falling back to iTerm2's own default profile name.
func DefaultProfile() string {
	if profile := os.Getenv("ITERM_PROFILE"); profile != "" {
		return profile
	}
	return "Default"
}

type session struct {
	c  *client.Client
	id string
}

func (s *session) SendText(t string) error {
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SendTextRequest{
			SendTextRequest: &api.SendTextRequest{
				Session: &s.id,
				Text:    &t,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error sending text to session %q: %w", s.id, err)
	}
	if status := resp.GetSendTextResponse().GetStatus(); status != api.SendTextResponse_OK {
		return fmt.Errorf("unexpected status for session %q: %s", s.id, status)
	}
	return nil
}

func (s *session) Activate(selectTab, orderWindowFront bool) error {
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: &api.ActivateRequest{
				Identifier: &api.ActivateRequest_SessionId{
					SessionId: s.id,
				},
				SelectTab:        &selectTab,
				OrderWindowFront: &orderWindowFront,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error activating session %q: %w", s.id, err)
	}
	if status := resp.GetActivateResponse().GetStatus(); status != api.ActivateResponse_OK {
		return fmt.Errorf("unexpected status for activate request: %s", status)
	}
	return nil
}

func (s *session) SplitPane(opts SplitPaneOptions) (Session, error) {
	direction := api.SplitPaneRequest_HORIZONTAL.Enum()
	if opts.Vertical {
		direction = api.SplitPaneRequest_VERTICAL.Enum()
	}
	profile := opts.Profile
	if profile == "" {
		profile = DefaultProfile()
	}
	resp, err := s.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SplitPaneRequest{
			SplitPaneRequest: &api.SplitPaneRequest{
				Session:        &s.id,
				SplitDirection: direction,
				ProfileName:    &profile,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error splitting pane: %w", err)
	}
	spResp := resp.GetSplitPaneResponse()
	if status := spResp.GetStatus(); status != api.SplitPaneResponse_OK {
		return nil, fmt.Errorf("could not split session %q with profile %q: %s", s.id, profile, status)
	}
	if len(spResp.GetSessionId()) < 1 {
		return nil, fmt.Errorf("expected at least one new session in split pane")
	}
	return &session{
		c:  s.c,
		id: spResp.GetSessionId()[0],
	}, nil
}

func (s *session) GetSessionID() string {
	return s.id
}
