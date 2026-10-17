// GP-SDR adapter for boatbod OP25. GPL-3.0-or-later, like the containing decoder.
// This header has no GNU Radio dependency so its exact wire format is testable.
#ifndef GPSDR_AUDIO_TAG_H
#define GPSDR_AUDIO_TAG_H
#include <stdint.h>
#include <stddef.h>
#include <string.h>
#include <vector>

class gpsdr_audio_tag {
    uint32_t sequence = 0, call = 0, group = 0, source = 0;
    uint16_t nac = 0;
    int8_t slot = -1;
    uint8_t algorithm = 0;
    bool known = false;
    std::vector<uint8_t> buffer;
    void u32(size_t at, uint32_t n) { for (int i=0;i<4;i++) buffer[at+i]=(n>>(8*i))&255; }
public:
    void context(uint32_t g,uint32_t s,uint16_t n,int8_t sl,uint8_t alg,bool k) {
        if (g!=group || s!=source || n!=nac || sl!=slot || alg!=algorithm || k!=known) ++call;
        group=g; source=s; nac=n; slot=sl; algorithm=alg; known=k;
    }
    // kind: 0=PCM, 1=DRAIN, 2=DROP. Every packet has its own sequence/identity.
    const std::vector<uint8_t>& packet(uint8_t kind,const void* pcm,size_t bytes) {
        buffer.resize(32+bytes);
        memcpy(buffer.data(),"GPSDP25\0",8);
        buffer[8]=1; buffer[9]=kind;
        uint16_t flags=(group?1:0)|(known?2:0)|((algorithm!=0x80)?4:0);
        buffer[10]=flags&255;buffer[11]=flags>>8;
        u32(12,++sequence);u32(16,call);u32(20,group);u32(24,source);
        buffer[28]=nac&255;buffer[29]=nac>>8;buffer[30]=uint8_t(slot);buffer[31]=algorithm;
        if(bytes) memcpy(buffer.data()+32,pcm,bytes);
        return buffer;
    }
    void end() { ++call; group=source=0;known=false; }
};
#endif
