package livekit

import (
	"context"
	"fmt"
	"time"

	lkproto "github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/auth"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// Client wraps the LiveKit server SDK for room and token management.
type Client struct {
	roomSvc *lksdk.RoomServiceClient
	apiKey  string
	secret  string
}

func NewClient(url, apiKey, apiSecret string) *Client {
	roomSvc := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	return &Client{
		roomSvc: roomSvc,
		apiKey:  apiKey,
		secret:  apiSecret,
	}
}

func (c *Client) CreateRoom(ctx context.Context, roomName string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.roomSvc.CreateRoom(ctx, &lkproto.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    60,
		MaxParticipants: 2,
	})
	if err != nil {
		return fmt.Errorf("livekit create room: %w", err)
	}
	return nil
}

func (c *Client) GenerateToken(roomName, identity, displayName string) (string, error) {
	grant := &auth.VideoGrant{
		Room:     roomName,
		RoomJoin: true,
	}

	token, err := auth.NewAccessToken(c.apiKey, c.secret).
		AddGrant(grant).
		SetIdentity(identity).
		SetName(displayName).
		SetValidFor(time.Hour).
		ToJWT()
	if err != nil {
		return "", fmt.Errorf("livekit generate token: %w", err)
	}
	return token, nil
}

func (c *Client) DeleteRoom(ctx context.Context, roomName string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.roomSvc.DeleteRoom(ctx, &lkproto.DeleteRoomRequest{
		Room: roomName,
	})
	if err != nil {
		return fmt.Errorf("livekit delete room: %w", err)
	}
	return nil
}
