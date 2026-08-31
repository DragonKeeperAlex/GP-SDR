// Read-only libusb inventory for matching SDRTrunk bus/port tuner identities.
// Uses libusb's public C ABI; no interface claims, reset, or RF operations.
#include <cstdint>
#include <cstdio>
#include <dlfcn.h>
#include <sstream>
#include <string>
#include <sys/types.h>
struct libusb_context;
struct libusb_device;
struct libusb_device_handle;
struct Descriptor {
    uint8_t length,type; uint16_t usb; uint8_t deviceClass,subClass,protocol,maxPacket;
    uint16_t vendor,product,device; uint8_t manufacturer,productString,serial,configurations;
};
int main() {
    void* library=nullptr;
    for (auto path : {"/opt/homebrew/lib/libusb-1.0.dylib","/usr/local/lib/libusb-1.0.dylib","libusb-1.0.dylib","libusb-1.0.so.0"}) {
        library=dlopen(path,RTLD_NOW|RTLD_LOCAL);if(library)break;
    }
    if(!library)return 1;
#define API(name, result, ...) auto name=reinterpret_cast<result(*)(__VA_ARGS__)>(dlsym(library,"libusb_" #name));if(!name)return 2
    API(init,int,libusb_context**); API(exit,void,libusb_context*);
    API(get_device_list,ssize_t,libusb_context*,libusb_device***);
    API(free_device_list,void,libusb_device**,int);
    API(get_device_descriptor,int,libusb_device*,Descriptor*);
    API(get_bus_number,uint8_t,libusb_device*);
    API(get_port_numbers,int,libusb_device*,uint8_t*,int);
    API(open,int,libusb_device*,libusb_device_handle**);
    API(close,void,libusb_device_handle*);
    API(get_string_descriptor_ascii,int,libusb_device_handle*,uint8_t,unsigned char*,int);
    libusb_context* context=nullptr;if(init(&context))return 3;
    libusb_device** devices=nullptr;auto count=get_device_list(context,&devices);
    for(ssize_t i=0;i<count;i++) {
        Descriptor d{};if(get_device_descriptor(devices[i],&d))continue;
        const char* kind=d.vendor==0x1d50&&d.product==0x6089?"HackRF":d.vendor==0x0bda&&(d.product==0x2838||d.product==0x2832)?"RTL-SDR":nullptr;
        if(!kind)continue;
        uint8_t ports[8]{};int n=get_port_numbers(devices[i],ports,8);if(n<=0)continue;
        std::ostringstream address;for(int p=0;p<n;p++){if(p)address<<'.';address<<int(ports[p]);}
        unsigned char serial[256]{};libusb_device_handle* handle=nullptr;
        if(!open(devices[i],&handle)){if(d.serial)get_string_descriptor_ascii(handle,d.serial,serial,255);close(handle);}
        // TSV fields are constrained so USB string descriptors cannot add rows.
        for(auto& c:serial)if(c=='\t'||c=='\n'||c=='\r')c=' ';
        printf("%s\t%s\t%u\t%s\n",kind,serial,get_bus_number(devices[i]),address.str().c_str());
    }
    if(devices)free_device_list(devices,1);exit(context);dlclose(library);return 0;
}
