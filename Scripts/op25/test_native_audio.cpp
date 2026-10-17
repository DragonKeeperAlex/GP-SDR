#include "gpsdr_audio_tag.h"
#include <cassert>
#include <cstdio>
static uint32_t u32(const std::vector<uint8_t>& p,int at){return p[at]|uint32_t(p[at+1])<<8|uint32_t(p[at+2])<<16|uint32_t(p[at+3])<<24;}
int main(){
 gpsdr_audio_tag t;int16_t pcm[2]={1,-2};
 t.context(100,200,0x123,-1,0x80,true);
 auto a=t.packet(0,pcm,sizeof(pcm));assert(a.size()==36 && a[10]==3 && a[30]==255 && u32(a,20)==100 && a[32]==1 && a[34]==254);auto call=u32(a,16);
 auto b=t.packet(0,pcm,sizeof(pcm));assert(u32(b,16)==call && u32(b,12)==2);
 auto end=t.packet(1,nullptr,0);assert(end.size()==32 && end[9]==1 && u32(end,16)==call);t.end();
 t.context(100,200,0x123,-1,0x80,true);auto next=t.packet(0,pcm,sizeof(pcm));assert(u32(next,16)!=call);
 t.context(100,200,0x123,1,0x84,true);auto encrypted=t.packet(0,pcm,sizeof(pcm));assert(encrypted[10]==7 && encrypted[30]==1);
 t.context(0,0,0,0,0x80,false);auto unknown=t.packet(0,pcm,sizeof(pcm));assert(unknown[10]==0);
 // One exact C++-encoded packet, independently consumed by Go transport tests.
 for(auto c:a) printf("%02x",c);printf("\n");
}
