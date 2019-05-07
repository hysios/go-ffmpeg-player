package player

import (
	"testing"
)

func TestPlayer_Open(t *testing.T) {
	type fields struct {
	}
	type args struct {
		filename string
	}

	type test struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	tests := []test{
		test{name: "open filename", fields: fields{}, args: args{"../videos/10.mp4"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var player Player
			if err := player.Open(tt.args.filename, nil); (err != nil) != tt.wantErr {
				t.Errorf("Player.Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlayer_Play(t *testing.T) {
	player, err := Open("../videos/10.mp4", nil)
	if err != nil {
		t.Errorf("Open %s", err)
	}

	player.PreFrame(func(frame *Frame) {
		t.Logf("frame %v", frame)
	})	

	if err := player.Play(); err != nil {
		t.Errorf("player %s", err)
	}

	// player.Wait()
}
