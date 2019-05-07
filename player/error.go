package player

//#cgo pkg-config: libavutil
//#include <libavutil/error.h>
//#include <stdlib.h>
//static const char *error2string(int code) { return av_err2str(code); }
import "C"
import "errors"

const (
	ErrEOF    = -('E' | ('O' << 8) | ('F' << 16) | (' ' << 24))
	ErrEAGAIN = -35
)

func ErrorFromCode(code C.int) error {
	if code >= 0 {
		return nil
	}

	return errors.New(C.GoString(C.error2string(code)))
}