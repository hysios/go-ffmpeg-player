#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/avutil.h>
#include <libavresample/avresample.h>
#include <libavutil/opt.h>
#include <libswscale/swscale.h>
#include <libavutil/parseutils.h>

typedef struct SwsImage {
    uint8_t *data[AV_NUM_DATA_POINTERS];
    int linesize[AV_NUM_DATA_POINTERS];
} SwsImage;

typedef struct player {
    AVFormatContext *fmt_ctx;
    AVOutputFormat *ofmt;
    AVPacket pkt;
    int stream_index;
    int *stream_mapping;
    int stream_mapping_size;
    int video_stream_index;
    AVCodec *dec;
    SwsImage sws_image;
    struct SwsContext *sws_ctx;
    AVCodecContext *dec_ctx;
    AVFrame *frame;
    AVRational time_base;   
} player;

int open_player(const char *filename, player *ply);
void close_player(player *ply);
int get_video_stream(player *ply);
int player_open_codec(player *ply);
void player_free_codec(player *ply);
int player_read_frame(player *ply, AVPacket* pkt);
int player_seek_frame(player *ply, int stream_idx, int64_t seek, int flags);
void free_packet(AVPacket* pkt);
int player_send_packet(player *ply, AVPacket* pkt);
int player_receive_frame(player *ply, AVFrame* frame);
int player_swscreate(player *ply, 
    int src_w, int src_h, 
    int dst_w, int dst_h, 
    enum AVPixelFormat src_pix_fmt, 
    enum AVPixelFormat dst_pix_fmt, 
    int flags);
void player_closesws(player *ply);
AVRational player_time_base(player *ply, int idx);
int player_hassws(player *ply);
int player_sws_scale(player *ply, 
    const uint8_t **src_data, 
    const int *src_linesize, 
    int src_w, int src_h, 
    int dst_w, int dst_h);
int player_alloc_swsimage(player *ply, int dst_w, int dst_h, enum AVPixelFormat dst_pix_fmt, int align);
void player_free_swsimage(player *ply);