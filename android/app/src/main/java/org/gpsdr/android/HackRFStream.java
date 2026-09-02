package org.gpsdr.android;

import android.hardware.usb.*;
import org.json.JSONObject;
import java.io.IOException;
import java.nio.*;
import java.util.*;
import java.util.concurrent.atomic.AtomicBoolean;
import org.gpsdr.engine.mobilebridge.USBStream;

/** Receive-only implementation of the documented libhackrf USB protocol.
 * Eight queued transfers avoid per-sample JNI calls. Never enables transmit.
 */
final class HackRFStream implements USBStream {
    final String deviceID;
    private UsbDeviceConnection connection;
    private UsbInterface usbInterface;
    private final List<UsbRequest> requests=new ArrayList<>();
    private final AtomicBoolean closed=new AtomicBoolean();
    private static final int[] FILTERS={1750000,2500000,3500000,5000000,5500000,6000000,7000000,8000000,9000000,10000000,12000000,14000000,15000000,20000000,24000000,28000000};
    HackRFStream(UsbManager manager,UsbDevice device,JSONObject spec) throws Exception {
        deviceID=device.getDeviceName();
        int rate=spec.getInt("SampleRateHz");
        long frequency=spec.getLong("CenterFrequencyHz");
        int lna=spec.optInt("LNAGainDB",16),vga=spec.optInt("VGAGainDB",16);
        if(rate<8000000 || rate>20000000) throw new IOException("Android HackRF capture requires 8–20 MS/s");
        if(frequency<1000000 || frequency>6000000000L) throw new IOException("HackRF frequency outside 1–6000 MHz");
        if(lna<0||lna>40||lna%8!=0||vga<0||vga>62||vga%2!=0) throw new IOException("Invalid HackRF gain steps");
        frequency=Math.round(frequency*(1.0+spec.optInt("PPMCorrection",0)/1e6));
        try {
            connection=manager.openDevice(device);
            if(connection==null) throw new IOException("Android could not open HackRF");
            UsbEndpoint input=null;
            for(int i=0;i<device.getInterfaceCount() && input==null;i++) {
                UsbInterface candidate=device.getInterface(i);
                for(int j=0;j<candidate.getEndpointCount();j++) {
                    UsbEndpoint ep=candidate.getEndpoint(j);
                    if(ep.getType()==UsbConstants.USB_ENDPOINT_XFER_BULK && ep.getDirection()==UsbConstants.USB_DIR_IN) { input=ep; usbInterface=candidate; break; }
                }
            }
            if(input==null||!connection.claimInterface(usbInterface,true)) throw new IOException("Cannot claim HackRF receive interface");
            out(1,0,0,null); // off while configuring
            out(6,0,0,words(rate,1));
            int filter=FILTERS[0]; for(int value:FILTERS) { if(value<=rate*.75) filter=value; }
            out(7,filter&0xffff,filter>>>16,null);
            out(16,0,0,words((int)(frequency/1000000),(int)(frequency%1000000)));
            out(17,spec.optBoolean("AmpEnabled",false)?1:0,0,null);
            out(23,spec.optBoolean("AntennaPower",false)?1:0,0,null);
            gain(19,lna); gain(20,vga);
            for(int i=0;i<8;i++) {
                UsbRequest request=new UsbRequest();
                if(!request.initialize(connection,input)) throw new IOException("Cannot initialize USB transfer");
                ByteBuffer buffer=ByteBuffer.allocateDirect(262144);
                request.setClientData(buffer); requests.add(request);
                if(!request.queue(buffer)) throw new IOException("Cannot queue USB receive buffer");
            }
            out(1,1,0,null); // receive only
        } catch(Exception error) { close(); throw error; }
    }
    private static byte[] words(int a,int b) { return ByteBuffer.allocate(8).order(ByteOrder.LITTLE_ENDIAN).putInt(a).putInt(b).array(); }
    private void out(int request,int value,int index,byte[] data) throws IOException {
        int length=data==null?0:data.length;
        if(connection.controlTransfer(0x40,request,value,index,data,length,1500)!=length) throw new IOException("HackRF USB control failed ("+request+")");
    }
    private void gain(int request,int gain) throws IOException {
        byte[] response=new byte[1];
        if(connection.controlTransfer(0xc0,request,0,gain,response,1,1500)!=1 || response[0]==0) throw new IOException("HackRF rejected gain");
    }
    @Override public byte[] readBlock() throws Exception {
        if(closed.get()) return new byte[0];
        UsbRequest request=connection.requestWait(3000);
        if(request==null) throw new IOException("HackRF stopped delivering USB samples");
        ByteBuffer buffer=(ByteBuffer)request.getClientData();
        int count=buffer.position();
        if(count<=0 || (count&1)!=0) throw new IOException("Invalid HackRF IQ transfer length");
        buffer.flip(); byte[] result=new byte[count]; buffer.get(result); buffer.clear();
        if(!closed.get() && !request.queue(buffer)) throw new IOException("HackRF disconnected during capture");
        return result;
    }
    boolean isClosed() { return closed.get(); }
    @Override public void close() {
        if(!closed.compareAndSet(false,true)) return;
        if(connection==null) return;
        try { out(1,0,0,null); out(23,0,0,null); out(17,0,0,null); } catch(Exception ignored) {}
        for(UsbRequest request:requests) { request.cancel(); request.close(); }
        if(usbInterface!=null) connection.releaseInterface(usbInterface);
        connection.close();
    }
}
