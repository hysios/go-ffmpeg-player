#include "player.h"

#include "libavutil/pixdesc.h"
#include "libavutil/imgutils.h"


int open_player(const char *filename, player *ply) {
    int ret = 0;
    
    if ((ret = avformat_open_input(&ply->fmt_ctx, filename, NULL, NULL)) < 0) {
        fprintf(stderr, "Could not open source file %s\n", filename);
        return ret;
    }

    ret = avformat_find_stream_info(ply->fmt_ctx, NULL);
    if (ret < 0) {
        fprintf(stderr, "Could not find stream information\n");
        return ret;
    }

    ply->sws_ctx = NULL;
    // ply->sws_image = NULL;

    av_dump_format(ply->fmt_ctx, 0, filename, 0);
    return 0;
}

void close_player(player *ply) {
    // avformat_close_input(&ply->fmt_ctx);
    avcodec_close(ply->dec_ctx);
    avcodec_free_context(&ply->dec_ctx);
    if (ply->frame != NULL) {
    	av_frame_free(&ply->frame);
    }

	if (ply->sws_ctx != NULL) {
		sws_freeContext(ply->sws_ctx);
	}    
}

int get_video_stream(player *ply) {
    int ret;
    ret = av_find_best_stream(ply->fmt_ctx, AVMEDIA_TYPE_VIDEO, -1, -1, &ply->dec, 0);
    if (ret < 0) {
        return ret;
    }
    ply->video_stream_index = ret;
    return ret;
}

int player_open_codec(player *ply) {
    int ret = 0;
    ply->dec_ctx = avcodec_alloc_context3(ply->dec);
    if (!ply->dec_ctx)
        return AVERROR(ENOMEM);
    avcodec_parameters_to_context(ply->dec_ctx, ply->fmt_ctx->streams[ply->video_stream_index]->codecpar);

    /* init the video decoder */
    if ((ret = avcodec_open2(ply->dec_ctx, ply->dec, NULL)) < 0) {
        av_log(NULL, AV_LOG_ERROR, "Cannot open video decoder\n");
        return ret;
    }

    return ret;
}

int player_read_frame(player *ply, AVPacket* pkt) {
    return av_read_frame(ply->fmt_ctx, pkt);
}

int player_seek_frame(player *ply, int stream_idx, int64_t seek, int flags ) {
    return av_seek_frame(ply->fmt_ctx, stream_idx, seek, flags);
}

void player_free_codec(player *ply) {
    // av_free(ply->dec);
    avcodec_close(ply->dec_ctx);
}

int player_swscreate(player *ply, 
    int src_w, int src_h, 
    int dst_w, int dst_h, 
    enum AVPixelFormat src_pix_fmt, 
    enum AVPixelFormat dst_pix_fmt, 
    int flags) {
    /* create scaling context */
    ply->sws_ctx = sws_getContext(src_w, src_h, src_pix_fmt,
                             dst_w, dst_h, dst_pix_fmt,
                             flags, NULL, NULL, NULL);
    if (!ply->sws_ctx) {
        fprintf(stderr,
            "Impossible to create scale context for the conversion "
            "fmt:%s s:%dx%d -> fmt:%s s:%dx%d\n",
            av_get_pix_fmt_name(src_pix_fmt), src_w, src_h,
            av_get_pix_fmt_name(dst_pix_fmt), dst_w, dst_h);        
        return AVERROR(EINVAL);
    }

    return player_alloc_swsimage(ply, dst_w, dst_h, dst_pix_fmt, 16);
}

void player_closesws(player *ply) {
    sws_freeContext(ply->sws_ctx);
    player_free_swsimage(ply);
}

int player_hassws(player *ply) {
    if (ply->sws_ctx != NULL) {
        return 1;
    } else {
        return 0;
    }
}

        /* convert to destination format */
int player_sws_scale(player *ply, 
    const uint8_t **src_data, 
    const int *src_linesize, 
    int src_w, int src_h, 
    int dst_w, int dst_h
) {
    struct SwsImage *sws_image = &ply->sws_image;
    uint8_t **dst_data = &sws_image->data[0];
    int *dst_linesize = &sws_image->linesize[0];

    // *dst_linesize = &linesize;
        /* convert to destination format */
    // printf("sws_scale ctx: %p, src_data %p, src_linesize: %p\n", ply->sws_ctx, src_data, src_linesize);
    // printf("                 , dst_data %p, dst_linesize: %p\n", dst_data, dst_linesize);
    // printf("sws_image %p, data: %p, linesize %p \n", sws_image, sws_image->data, sws_image->linesize);

    return sws_scale(ply->sws_ctx, src_data,
            src_linesize, 0, src_h, sws_image->data, sws_image->linesize);
}


void free_packet(AVPacket* pkt) {
    av_packet_unref(pkt);
}

int player_send_packet(player *ply, AVPacket* pkt) {
    return avcodec_send_packet(ply->dec_ctx, pkt);
}

int player_receive_frame(player *ply, AVFrame* frame) {
    return avcodec_receive_frame(ply->dec_ctx, frame);
}

AVRational player_time_base(player *ply, int idx) {
    return ply->fmt_ctx->streams[idx]->time_base;
}

int player_alloc_swsimage(player *ply, int dst_w, int dst_h, enum AVPixelFormat dst_pix_fmt, int align) {
    int ret;

    // if (ply->sws_image == NULL) {
    //     ply->sws_image = malloc(sizeof(ply->sws_image));
    // }
    SwsImage *sws_image = &ply->sws_image;

    if ((ret = av_image_alloc(sws_image->data, sws_image->linesize,
                              dst_w, dst_h, dst_pix_fmt, align)) < 0) {
        fprintf(stderr, "Could not allocate destination image\n");
        return ret;
    }

    return 0;
}

void player_free_swsimage(player *ply) {
    // if (ply->sws_image != NULL) {
    SwsImage *sws_image = &ply->sws_image;
    av_freep(sws_image->data);
        // free(ply->sws_image);
        // ply->sws_image = NULL;
    // }
}