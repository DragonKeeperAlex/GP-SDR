package org.gpsdr.android;

import android.app.PendingIntent;
import android.content.*;
import android.hardware.usb.*;
import android.os.Build;
import android.util.Log;
import org.json.*;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import org.gpsdr.engine.mobilebridge.USBHost;
import org.gpsdr.engine.mobilebridge.USBStream;

final class AndroidUSBHost implements USBHost, AutoCloseable {
    private final Context context;
    private final UsbManager manager;
    private final Runnable changed;
    private final Set<HackRFStream> streams = ConcurrentHashMap.newKeySet();
    private static final String PERMISSION = "org.gpsdr.android.USB_PERMISSION";
    private final BroadcastReceiver receiver = new BroadcastReceiver() {
        @Override public void onReceive(Context c, Intent intent) { changed.run(); }
    };
    AndroidUSBHost(Context c, Runnable changed) {
        context=c; this.changed=changed; manager=c.getSystemService(UsbManager.class);
        IntentFilter filter=new IntentFilter(PERMISSION);
        filter.addAction(UsbManager.ACTION_USB_DEVICE_ATTACHED);
        filter.addAction(UsbManager.ACTION_USB_DEVICE_DETACHED);
        if(Build.VERSION.SDK_INT>=33) c.registerReceiver(receiver,filter,Context.RECEIVER_NOT_EXPORTED);
        else c.registerReceiver(receiver,filter);
    }
    private boolean hackrf(UsbDevice d) { return d.getVendorId()==0x1d50 && d.getProductId()==0x6089; }
    private boolean rtl(UsbDevice d) { return d.getVendorId()==0x0bda && (d.getProductId()==0x2838 || d.getProductId()==0x2832); }
    public void requestPermissions() {
        for(UsbDevice d:manager.getDeviceList().values()) if(hackrf(d) && !manager.hasPermission(d)) {
            Intent intent=new Intent(PERMISSION).setPackage(context.getPackageName());
            int flags=PendingIntent.FLAG_UPDATE_CURRENT | (Build.VERSION.SDK_INT>=31?PendingIntent.FLAG_MUTABLE:0);
            manager.requestPermission(d,PendingIntent.getBroadcast(context,d.getDeviceId(),intent,flags));
        }
    }
    @Override public String devicesJSON() throws Exception {
        JSONArray items=new JSONArray(); int hacks=0, rtls=0;
        List<UsbDevice> devices=new ArrayList<>(manager.getDeviceList().values());
        devices.sort(Comparator.comparing(UsbDevice::getDeviceName));
        for(UsbDevice d:devices) {
            boolean hack=hackrf(d); if(!hack && !rtl(d)) continue;
            boolean permitted=manager.hasPermission(d);
            JSONObject item=new JSONObject();
            item.put("id",d.getDeviceName()); item.put("kind",hack?"HackRF":"RTL-SDR");
            item.put("name",(hack?"HackRF One "+(++hacks):"RTL-SDR "+(++rtls)));
            item.put("connected",true); item.put("available",hack&&permitted);
            item.put("sampleRateLimit",hack?20e6:2.4e6);
            item.put("note",!hack?"Direct Android RTL driver is not included in this preview. RTL-TCP is supported.":permitted?"Android USB receive transport • RF validation pending":"Tap USB in the top bar to grant access");
            if(permitted) { try { item.put("serial",d.getSerialNumber()); } catch(SecurityException ignored) {} }
            items.put(item);
        }
        String result=items.toString();
        Log.i("GP-SDR-USB","enumerated "+items.length()+" receiver(s): "+result);
        return result;
    }
    @Override public USBStream open(String id,String specification) throws Exception {
        UsbDevice device=manager.getDeviceList().get(id);
        if(device==null) throw new java.io.IOException("Receiver disconnected");
        if(!manager.hasPermission(device)) throw new java.io.IOException("Tap USB to grant Android receiver access");
        if(!hackrf(device)) throw new java.io.IOException("Direct RTL USB is not implemented in this preview; use RTL-TCP");
        for(HackRFStream stream:streams) if(!stream.isClosed() && stream.deviceID.equals(id)) throw new java.io.IOException("This USB receiver is already in use");
        HackRFStream stream=new HackRFStream(manager,device,new JSONObject(specification));
        streams.removeIf(HackRFStream::isClosed); streams.add(stream); return stream;
    }
    @Override public void close() { context.unregisterReceiver(receiver); for(HackRFStream stream:streams) stream.close(); streams.clear(); }
}
