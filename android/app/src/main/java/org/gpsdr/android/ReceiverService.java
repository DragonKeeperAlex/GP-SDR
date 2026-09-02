package org.gpsdr.android;

import android.app.*;
import android.content.*;
import android.os.*;
import java.util.concurrent.*;
import org.gpsdr.engine.mobilebridge.Engine;
import org.gpsdr.engine.mobilebridge.Mobilebridge;

/** Owns the receiver independently of Activity rotation and screen sleep. */
public final class ReceiverService extends Service {
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final IBinder binder = new LocalBinder();
    private volatile Engine engine;
    private volatile String error = "";
    private AndroidUSBHost usb;
    private PowerManager.WakeLock wake;
    public final class LocalBinder extends Binder { ReceiverService getService() { return ReceiverService.this; } }
    public String url() { return engine == null ? "" : engine.url(); }
    public String error() { return error; }
    public void refreshUSB() { worker.execute(() -> { if (engine != null) engine.refreshDevices(); }); }
    public void requestUSB() { usb.requestPermissions(); }
    @Override public void onCreate() {
        super.onCreate();
        NotificationManager notifications = getSystemService(NotificationManager.class);
        notifications.createNotificationChannel(new NotificationChannel("receiver", "Receiver", NotificationManager.IMPORTANCE_LOW));
        PendingIntent open = PendingIntent.getActivity(this, 0, new Intent(this, MainActivity.class), PendingIntent.FLAG_IMMUTABLE);
        Notification notification = new Notification.Builder(this, "receiver")
            .setContentTitle("GP-SDR receiver")
            .setContentText("Local engine running • open GP-SDR to stop")
            .setSmallIcon(android.R.drawable.ic_menu_compass).setContentIntent(open).setOngoing(true).build();
        startForeground(1, notification);
        wake = getSystemService(PowerManager.class).newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "gpsdr:receiver");
        wake.acquire();
        usb = new AndroidUSBHost(this, this::refreshUSB);
        worker.execute(() -> {
            try { engine = Mobilebridge.start(Storage.prepare(this).getAbsolutePath(), usb); }
            catch (Exception e) { error = e.getMessage() == null ? e.toString() : e.getMessage(); if(wake.isHeld()) wake.release(); }
        });
    }
    @Override public int onStartCommand(Intent intent, int flags, int id) { return START_NOT_STICKY; }
    @Override public IBinder onBind(Intent intent) { return binder; }
    @Override public void onDestroy() {
        usb.close();
        worker.execute(() -> { if(engine != null) engine.stop(); if(wake.isHeld()) wake.release(); });
        worker.shutdown();
        stopForeground(STOP_FOREGROUND_REMOVE);
        super.onDestroy();
    }
}
