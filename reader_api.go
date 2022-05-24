package imghash

// An unfinished implementation using libav* libraries, scrapped because I realized
// that hardware acceleration and general speed improvements were a pain in the ass
// to implement. Left here because it was hard to do and I'm mildly proud of it.

/*

#cgo pkg-config: libavcodec libavformat libavutil libswscale

#include "libavcodec/avcodec.h"
#include "libavformat/avformat.h"
#include "libavutil/imgutils.h"
#include "libswscale/swscale.h"

static AVStream* get_stream(AVFormatContext *ctx, int idx) {
	return ctx->streams[idx];
}

void free_frame(AVFrame *frame) {
	av_freep(&frame->data[0]);
}

static int eof_err(int n) {
	return n == AVERROR_EOF || n == AVERROR(EAGAIN);
}

*/
import "C"

import (
	"bytes"
	"image"
	"unsafe"

	"github.com/pkg/errors"
)

func AvErrorStr(averr int) error {
	errlen := 1024
	b := make([]byte, errlen)

	C.av_strerror(C.int(averr), (*C.char)(unsafe.Pointer(&b[0])), C.size_t(errlen))

	return errors.New(string(b[:bytes.IndexByte(b, 0)]))

}

func NewFromVideo2(name string) (*File, error) {
	nm := C.CString(name)
	defer C.free(unsafe.Pointer(nm))

	var (
		inputCtx *C.AVFormatContext
		dec      *C.AVCodec
		ret      C.int
	)

	// Open the file and verify stream info
	if ret := C.avformat_open_input(&inputCtx, nm, nil, nil); ret < 0 {
		return nil, errors.New("Could not create file context")
	}
	defer C.avformat_close_input(&inputCtx)

	if ret := C.avformat_find_stream_info(inputCtx, nil); ret < 0 {
		return nil, errors.New("Could not find stream information")
	}

	//ctx.start_time = 0

	// select the video stream
	streamIdx := C.av_find_best_stream(inputCtx, C.AVMEDIA_TYPE_VIDEO, -1, -1, &dec, 0)
	if streamIdx < 0 {
		return nil, errors.New("Could not find a video stream in the input file")
	}

	// create decoding context
	decCtx := C.avcodec_alloc_context3(dec)
	if decCtx == nil {
		return nil, AvErrorStr(C.ENOMEM)
	}
	defer C.avcodec_free_context(&decCtx)

	// setup experimental compliance
	if dec.capabilities&C.AV_CODEC_CAP_EXPERIMENTAL != 0 {
		decCtx.strict_std_compliance = C.FF_COMPLIANCE_EXPERIMENTAL
	}

	stream := C.get_stream(inputCtx, streamIdx)
	C.avcodec_parameters_to_context(decCtx, stream.codecpar)

	// init the video decoder
	if ret := C.avcodec_open2(decCtx, dec, nil); ret < 0 {
		return nil, errors.New("Could not open video decoder")
	}

	// create encoding context
	enc := C.avcodec_find_encoder(C.AV_CODEC_ID_RAWVIDEO)
	encCtx := C.avcodec_alloc_context3(enc)
	if encCtx == nil {
		return nil, errors.New("Could not allocate video context")
	}
	defer C.avcodec_free_context(&encCtx)

	encCtx.width = C.int(width)
	encCtx.height = C.int(height)
	encCtx.pix_fmt = C.AV_PIX_FMT_RGBA
	encCtx.time_base = C.AVRational{1, 1}
	//encCtx.time_base = C.AVRational{1, 12}
	//encCtx.framerate = C.AVRational{12, 1}

	// init the video encoder
	if C.avcodec_open2(encCtx, enc, nil) < 0 {
		return nil, errors.New("Could not open codec")
	}

	// setup the sws context
	swsCtx := C.sws_getContext(decCtx.width, decCtx.height, decCtx.pix_fmt, encCtx.width, encCtx.height, encCtx.pix_fmt, C.SWS_BILINEAR, nil, nil, nil)
	if swsCtx == nil {
		return nil, errors.New("Could not create sws context")
	}
	defer C.sws_freeContext(swsCtx)

	/*

		ACTUAL DECODING

	*/

	// Allocate necessary packets and frames
	packet := C.av_packet_alloc()
	outpacket := C.av_packet_alloc()
	frame := C.av_frame_alloc()
	outframe := C.av_frame_alloc()

	if packet == nil || outpacket == nil {
		return nil, errors.Errorf("formatting packets: packet: %v, outpacket: %v", packet, outpacket)
	}
	defer C.av_packet_free(&packet)
	defer C.av_packet_free(&outpacket)

	if frame == nil || outframe == nil {
		return nil, errors.Errorf("formatting frames: frame: %v, outframe: %v", packet, outpacket)
	}
	defer C.av_frame_free(&frame)
	defer C.av_frame_free(&outframe)

	outframe.width = encCtx.width
	outframe.height = encCtx.height
	outframe.format = C.int(encCtx.pix_fmt)

	if ret := C.av_image_alloc(
		(**C.uint8_t)(unsafe.Pointer(&outframe.data)),
		(*C.int)(unsafe.Pointer(&outframe.linesize)),
		outframe.width, outframe.height, int32(outframe.format), 32); ret < 0 {
		return nil, errors.Wrap(AvErrorStr(int(ret)), "allocating out image buffer")
	}
	defer C.free_frame(outframe)

	var img image.RGBA
	img.Stride = 4 * width
	img.Rect = image.Rect(0, 0, width, height)

	file := new(File)

	for {
		if ret = C.av_read_frame(inputCtx, packet); ret < 0 {
			break
		}

		if packet.stream_index != streamIdx {
			C.av_packet_unref(packet)
			continue
		}

		if ret = C.avcodec_send_packet(decCtx, packet); ret < 0 {
			return nil, errors.Wrap(AvErrorStr(int(ret)), "Error while sending a packet to the decoder")
		}

		for ret >= 0 {
			ret = C.avcodec_receive_frame(decCtx, frame)
			if C.eof_err(ret) == 1 {
				break
			} else if ret < 0 {
				return nil, errors.Wrap(AvErrorStr(int(ret)), "Error while receiving a frame from the decoder")
			}

			C.sws_scale(
				swsCtx,
				(**C.uint8_t)(unsafe.Pointer(&frame.data)),
				(*C.int)(unsafe.Pointer(&frame.linesize)),
				0,
				C.int(frame.height),
				(**C.uint8_t)(unsafe.Pointer(&outframe.data)),
				(*C.int)(unsafe.Pointer(&outframe.linesize)),
			)

			if ret = C.avcodec_send_frame(encCtx, outframe); ret < 0 {
				return nil, errors.Wrap(AvErrorStr(int(ret)), "sending resized frame to context")
			}

			for ret >= 0 {
				ret = C.avcodec_receive_packet(encCtx, outpacket)
				if C.eof_err(ret) == 1 {
					break
				}

				//log.Println(frame.pts, decCtx.frame_number, encCtx.frame_number, outpacket.size)
				img.Pix = C.GoBytes(unsafe.Pointer(outpacket.data), outpacket.size)

				vh, hh, err := differenceHash(&img)
				if err != nil {
					return nil, errors.Wrap(err, "creating hash")
				}

				file.hashes = append(file.hashes, Hash{
					VHash: vh,
					HHash: hh,
					Index: uint32(encCtx.frame_number),
				})

				C.av_packet_unref(outpacket)
			}

			C.av_frame_unref(frame)
		}

		C.av_packet_unref(packet)
	}
	return file, nil
}
