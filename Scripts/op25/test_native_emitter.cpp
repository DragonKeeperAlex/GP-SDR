// Synthetic socket-only exercise of the actual linked OP25 audio emitter.
#include "op25_audio.h"
#include <stdlib.h>
int main(int argc,char** argv){
 if(argc!=2)return 64;
 log_ts logger;
 op25_audio out("127.0.0.1",atoi(argv[1]),logger,0,0);
 int16_t pcm[2]={1,-2};
 out.set_gpsdr_context(100,200,0x123,-1,0x80,true);out.send_audio(pcm,sizeof(pcm));
 out.set_gpsdr_context(101,201,0x123,1,0x80,true);out.send_audio(pcm,sizeof(pcm));
 out.send_audio_flag(op25_audio::DRAIN);
 out.set_gpsdr_context(102,202,0x123,0,0x84,true);out.send_audio(pcm,sizeof(pcm));
 out.stop();
}
