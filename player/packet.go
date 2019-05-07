package player

// #include <libavcodec/avcodec.h>
import "C"

type Packet struct {
	packet C.AVPacket
}


func (pkt *Packet) StreamIndex() int {
	return int(pkt.packet.stream_index)
}