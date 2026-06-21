package gameactor

import "context"

type RoomRuntime struct {
	commands *RoomCommandRegistry
}

func NewRoomRuntime(buffer int) *RoomRuntime {
	return &RoomRuntime{commands: NewRoomCommandRegistry(buffer)}
}

func (r *RoomRuntime) RunRoomCommand(ctx context.Context, roomID string, eventType RoomEventType, lane EventLane, run func() error) error {
	if r == nil {
		if run == nil {
			return nil
		}
		return run()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return r.commands.Do(ctx, RoomEvent{
		RoomID: roomID,
		Type:   eventType,
		Lane:   lane,
	}, func(context.Context) error {
		if run == nil {
			return nil
		}
		return run()
	})
}

func (r *RoomRuntime) RemoveRoom(roomID string) {
	if r != nil && r.commands != nil {
		r.commands.RemoveRoom(roomID)
	}
}

func (r *RoomRuntime) Close() {
	if r != nil && r.commands != nil {
		r.commands.Close()
	}
}
