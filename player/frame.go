package player

// #include <libavutil/frame.h>
// #include <libavformat/avformat.h>
// #include <libavutil/avutil.h>
// #include "player.h"
import "C"

import (
	"unsafe"
	// "log"
	"errors"
)

type Frame struct {
	// frame *C.AVFrame
	pkt_dts int64
	PTS int64
	Data [][]byte
	Linesize []int
	Width int32
	Height int32
}

type Stream struct {
	stream *C.AVStream
	time_base *C.AVRational
}

var microseconds = C.AVRational{1, 1000000};


func FromFrame(frame *Frame, avframe *C.AVFrame) {
	data, linesize := frameDataLinesize((**C.uint8_t)(&avframe.data[0]), 
		(*C.int)(&avframe.linesize[0]), 
		int(avframe.width), int(avframe.height))

	frame.Data = data
	frame.Linesize = linesize
	frame.PTS = int64(avframe.pts)
	frame.pkt_dts = int64(avframe.pkt_dts)
	frame.Width = int32(avframe.width)
	frame.Height = int32(avframe.height)
}

func frameDataLinesize(data **C.uint8_t, linesize *C.int, width, height int) (dstData [][]byte, dstLinesize []int) {
	dstData = make([][]byte, 3)
	dstLinesize = make([]int, 3)
	wsize := uintptr(4)
	for i := uintptr(0); i < 3; i++ {

		vv := (**C.uint8_t)(unsafe.Pointer(uintptr(unsafe.Pointer(data)) + i * 8))
		size := (*C.int)(unsafe.Pointer(uintptr(unsafe.Pointer(linesize)) + i * wsize))
		
		dstLinesize[i] = int(*size)
		f := float64(*size) / float64(width)
		h := C.int(float64(height) * f)

		// log.Printf("gobytes size %d [%dx%d]\n", *size * h, *size,h )
		dstData[i] = C.GoBytes(unsafe.Pointer(*vv), *size * h)
		// log.Printf("gobytes address %p\n", &dstData[i][0])
	}
	return 
}

func playerSwsDataLinesize(ply *Player, width, height int) (dstData [][]byte, dstLinesize []int, ok bool) {
	player := ply.player
	// if player.sws_image == nil {
	// 	return nil, nil, false
	// }
	sws_image := player.sws_image

	dstData, dstLinesize = frameDataLinesize((**C.uint8_t)(&sws_image.data[0]),
		(*C.int)(&sws_image.linesize[0]), 
		width, height)
	return dstData, dstLinesize, true
}

func (frame *Frame) rescale_q(time_base C.AVRational) int64 {
    // const AVRational microseconds = {1, 1000000};

	return int64(C.av_rescale_q(
		C.int64_t(frame.pkt_dts),
		time_base,
		microseconds));
}

func ScaleFrame(ply *Player, srcframe *C.AVFrame, scale_w, scale_h int, frame *Frame) error {
	var (
		src_data = (**C.uint8_t)(&srcframe.data[0])
		src_linesize = (*C.int)(&srcframe.linesize[0])
	)

	// data := (**C.uint8_t)(&dst_data[0])
	// linesize := (*C.int)(&dst_linesize[0])


	ret := C.player_sws_scale(&ply.player, 
		src_data, src_linesize, 
		C.int(ply.width), C.int(ply.height),
		C.int(scale_w), C.int(scale_h),
	)
	if ret < 0 {
		return ErrorFromCode(ret) 
	}

	data, linesize, ok := playerSwsDataLinesize(ply, 
		ply.scaleWidth, ply.scaleHeight)
	if !ok {
		return errors.New("can't get Sws Data")
	}

	frame.Data = data
	frame.Linesize = linesize
	frame.PTS = int64(srcframe.pts)
	frame.pkt_dts = int64(srcframe.pkt_dts)
	frame.Width = int32(ply.scaleWidth)
	frame.Height = int32(ply.scaleHeight)

	// defer C.av_freep(unsafe.Pointer(&dst_data))

	return ErrorFromCode(ret)
}