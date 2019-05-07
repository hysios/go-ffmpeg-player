package player

//#cgo pkg-config: libavcodec libavformat libavutil libavresample libswscale
//#include "player.h"
import "C"

import (
	"io"
	"time"
	"sync"
	"log"
	"errors"
)

type delayFunc = func(*Player) error

type Player struct {
	sync.RWMutex
	sync.Once	

	Options Options
	// Frame Frame
	Status Status
	SeekPos int64
	VideoStreamIndex int
	SoundStreamIndex int
	FrameRender FrameRender

	queue []delayFunc
	player C.player
	frame *C.AVFrame
	width int 
	height int
	scaleWidth int
	scaleHeight int
}

type Options struct {
	AutoPlay bool
	Loop bool
	DumpFormat bool
	Initial bool
}

type Status int

const (
	StatePlaying Status = iota
	StatePause
	StateSeek
	StateEnd
)

type FrameRender func(frame *Frame) 

func Open(filename string, opt *Options) (*Player, error) {
	var player Player


	if err := player.Open(filename, opt); err != nil {
		return nil, err
	}


	return &player, nil
}

func (player *Player) Open(filename string, opt *Options) error {
	if opt != nil {
		player.Options = *opt
	}
	return ErrorFromCode(C.open_player(C.CString(filename), &player.player))
}

func (player *Player) init() {
	player.videoStream()
	player.openCodec()
	player.allocFrame()
	player.codecSize()
}

func (player *Player) Close() error {
	C.close_player(&player.player)
	C.av_frame_free(&player.frame)

	if player.player.sws_ctx != nil {
		C.player_closesws(&player.player)
	}
	// C.player_free_codec(&player.player)

	return nil
}

func (player *Player) videoStream() int {
	player.VideoStreamIndex = int(C.get_video_stream(&player.player))
	return player.VideoStreamIndex
}

func (player *Player) openCodec() error {
	return ErrorFromCode(C.player_open_codec(&player.player))
}

func (player *Player) codecSize() (width, height int, err error) {
	if player.player.dec_ctx == nil {
		return 0, 0, errors.New("player codec is not open")
	}
	width = int(player.player.dec_ctx.width)
	height = int(player.player.dec_ctx.height)
	player.width = width
	player.height = height
	return
}

func (player *Player) swscreate(src_w, src_h int, dst_w, dst_h int, pix_fmt int) error {
	return ErrorFromCode(C.player_swscreate(&player.player, 
		C.int(src_w),
		C.int(src_h),
		C.int(dst_w),
		C.int(dst_h),
		C.enum_AVPixelFormat(pix_fmt), C.enum_AVPixelFormat(pix_fmt),
		C.SWS_BILINEAR))
}

func (player *Player) allocFrame() {
	player.frame = C.av_frame_alloc()
}

func (player *Player) timeBase(idx int) C.AVRational {
	return  C.player_time_base(&player.player, C.int(idx))
}

func (player *Player) Play() error {
	player.Lock()
	defer player.Unlock()

	player.Status = StatePlaying

	return nil
}

func (player *Player) Pause() {
	player.Lock()
	defer player.Unlock()

	player.Status = StatePause
}

func (player *Player) Resume() {
	player.Lock()
	defer player.Unlock()

	player.Status = StatePlaying
}

func (player *Player) Seek(seek int64) {
	player.Lock()
	defer player.Unlock()
	player.Status = StateSeek
	player.SeekPos = seek
}

func (player *Player) Reset() error {
	player.Lock()
	defer player.Unlock()	
	player.Status = StateSeek
	player.SeekPos = 0
	return nil
}

func (player *Player) SetScale( width, height int) {
	player.queue = append(player.queue, func(player *Player) error {
		player._setScale(width, height)
		return nil
	})
}

func (player *Player) SetScaleSize( size string) {
	player.queue = append(player.queue, func(player *Player) error {
		return player._setScaleSize(size)
	})
}

func (player *Player) _setScale(width, height int) {
	// AV_PIX_FMT_YUV420P
	player.swscreate(player.width, player.height, width, height, C.AV_PIX_FMT_YUV420P)
	player.scaleWidth = width
	player.scaleHeight = height
}

func (player *Player) _setScaleSize(size string) error {
	// AV_PIX_FMT_YUV420P
	var width, height C.int
	ret := C.av_parse_video_size(&width, &height, C.CString(size))
	if ret < 0 {
		return ErrorFromCode(ret)
	}

	player.swscreate(player.width, player.height, int(width), int(height), C.AV_PIX_FMT_YUV420P)
	player.scaleWidth = int(width)
	player.scaleHeight = int(height)
	return nil
}

func (player *Player) HasScale() bool {
	return player.scaleWidth > 0 || player.scaleHeight > 0
}

func (player *Player) PreFrame(render FrameRender) {
	player.Lock()
	defer player.Unlock()	
	player.FrameRender = render
}

func (player *Player ) GetState() Status {
	player.RLock()
	defer player.RUnlock()
	return player.Status
}

func (player *Player) doqueue() {
	for _, fn := range player.queue {
		fn(player)
	}
}

func (player *Player) readPacket() (*Packet, error) {
	var packet Packet

	ret := C.player_read_frame(&player.player, &packet.packet)
	if ret == ErrEAGAIN || ret == ErrEOF {
		return nil, io.EOF
	} else if err := ErrorFromCode(ret); err != nil {
		return nil, err
	}
	
	return &packet, nil
}

func (player *Player) DecodeFrame(pkt *Packet, decodeCb func(frame *Frame)) error {
	// const AVRational microseconds = {1, 1000000};
	var (
		frame Frame
	)

	ret := C.player_send_packet(&player.player, &pkt.packet)
	if ret < 0 {
		return ErrorFromCode(ret)
	}

	for {
		ret = C.player_receive_frame(&player.player, player.frame);
		if ret == ErrEAGAIN || ret == ErrEOF {
			return io.EOF
		} else if (ret < 0) {
			return ErrorFromCode(ret)
		}

		if C.player_hassws(&player.player) > 0 {
			ScaleFrame(player, player.frame, player.scaleWidth, player.scaleHeight, &frame)
		} else {
			FromFrame(&frame, player.frame)
		}

		time_base := player.timeBase(player.VideoStreamIndex)
		// log.Printf("frame before PTS %d\n", frame.PTS)
		frame.PTS = frame.rescale_q(time_base)
		// log.Printf("frame after PTS %d\n", frame.PTS)

		if decodeCb != nil {
			decodeCb(&frame)
		}
	}
}

func (player *Player) decodeVideoFrame(pkt *Packet, decodeCb func(frame *Frame)) error {
	var (
		frame Frame
	)

	ret := C.player_send_packet(&player.player, &pkt.packet)
	if ret < 0 {
		return ErrorFromCode(ret)
	}

	for {
		ret = C.player_receive_frame(&player.player, player.frame);
		if ret == ErrEAGAIN || ret == ErrEOF {
			return io.EOF
		} else if (ret < 0) {
			return ErrorFromCode(ret)
		}
		
		FromFrame(&frame, player.frame)
		time_base := player.timeBase(player.VideoStreamIndex)
		// log.Printf("frame before PTS %d\n", frame.PTS)
		frame.PTS = frame.rescale_q(time_base)
		// log.Printf("frame after PTS %d\n", frame.PTS)

		if decodeCb != nil {
			decodeCb(&frame)
		}
	}
}

func (player *Player) Wait() error {
	var (
		pkt *Packet
		err error
	)
	player.Do(func() {
		player.init()
		player.doqueue()
		// width, height, _ := player.CodecSize()
		log.Printf("size %dx%d", player.width, player.height)
		for {
			switch player.GetState() {
			case StateSeek:
				seek := player.SeekPos
				// player.Seek(player.SeekPos)
				log.Printf("seek %d\n", seek)
				C.player_seek_frame(&player.player, 
					C.int(player.VideoStreamIndex), 
					C.int64_t(seek), 
					C.AVSEEK_FLAG_FRAME | C.AVSEEK_FLAG_ANY)
				player.Status = StatePlaying
				player.SeekPos = -1
				fallthrough
			case StatePlaying:
				pkt, err = player.checkPacket(player.readPacket())
				if err != nil {
					// log.Printf("readPacket %s", err)
					if player.Options.Loop {
						break
					} else {
						return
					}
				}

		        if pkt.StreamIndex() == player.VideoStreamIndex {
					player.DecodeFrame(pkt, player.FrameRender)
				}
			case StatePause, StateEnd:
				fallthrough
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	})

	return err
}

func (player *Player) checkPacket(pkt *Packet, err error) (*Packet, error) {
	if err == io.EOF {
		log.Printf("readPacket %s", err)
		player.Status = StateSeek
		player.SeekPos = 0
		return pkt, err
	} else if err != nil {
		return pkt, err
	}

	return pkt, nil
}