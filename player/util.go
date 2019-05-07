package player

//#include "player.h"
import "C"

func ParseVideoSize(size string) (width, height int, err error ) {
	var cw, ch C.int
	if ret := C.av_parse_video_size(&cw, &ch, C.CString(size)); ret < 0 {
		err = ErrorFromCode(ret)
	}

	width = int(cw)
	height = int(ch)

	return 
}