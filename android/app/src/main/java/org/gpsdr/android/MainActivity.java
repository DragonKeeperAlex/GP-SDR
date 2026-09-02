package org.gpsdr.android;

import android.Manifest;
import android.app.*;
import android.content.*;
import android.net.Uri;
import android.os.*;
import android.view.*;
import android.webkit.*;
import android.widget.*;
import java.io.*;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.*;
import java.util.concurrent.*;

public final class MainActivity extends Activity {
    private WebView web;
    private TextView state;
    private ReceiverService service;
    private boolean bound, started;
    private String origin="",download="";
    private final Handler main=new Handler(Looper.getMainLooper());
    private final ExecutorService io=Executors.newSingleThreadExecutor();
    private ValueCallback<Uri[]> fileCallback;
    private final ServiceConnection connection=new ServiceConnection() {
        @Override public void onServiceConnected(ComponentName name,IBinder binder) {
            service=((ReceiverService.LocalBinder)binder).getService(); poll();
        }
        @Override public void onServiceDisconnected(ComponentName name) { service=null; state.setText("Receiver service disconnected. Stop and restart to reconnect."); }
    };
    @Override public void onCreate(Bundle saved) {
        super.onCreate(saved);
        LinearLayout root=new LinearLayout(this);root.setOrientation(LinearLayout.VERTICAL);root.setBackgroundColor(0xff101318);
        LinearLayout toolbar=new LinearLayout(this);
        Button run=button("Start",()->{if(started){stop();runState();}else start();});run.setTag("run");toolbar.addView(run);
        toolbar.addView(button("USB",()->{if(service!=null)service.requestUSB();else message("Start the engine first, then grant USB access.");}));
        toolbar.addView(button("Performance",this::performance));
        toolbar.addView(button("Storage",this::storage));
        HorizontalScrollView strip=new HorizontalScrollView(this);strip.addView(toolbar);root.addView(strip);
        state=new TextView(this);state.setTextColor(0xffb4c0cc);state.setTextSize(12);state.setPadding(12,4,12,4);root.addView(state);
        web=new WebView(this);web.setBackgroundColor(0xff101318);
        web.getSettings().setJavaScriptEnabled(true);web.getSettings().setDomStorageEnabled(true);
        web.getSettings().setAllowFileAccess(false);web.getSettings().setAllowContentAccess(true);
        web.getSettings().setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        web.getSettings().setMediaPlaybackRequiresUserGesture(true);
        web.setWebViewClient(new WebViewClient(){
            @Override public boolean shouldOverrideUrlLoading(WebView view,WebResourceRequest request){
                if(local(request.getUrl()))return false;
                if(request.isForMainFrame() && ("https".equals(request.getUrl().getScheme())||"http".equals(request.getUrl().getScheme())))
                    new AlertDialog.Builder(MainActivity.this).setMessage("Open this external link in your browser?").setPositiveButton("Open",(d,w)->startActivity(new Intent(Intent.ACTION_VIEW,request.getUrl()))).setNegativeButton("Cancel",null).show();
                return true;
            }
            @Override public void onPageFinished(WebView view,String url){if(local(Uri.parse(url))){applyPerformance();applyMobileCapabilities();}}
        });
        web.setWebChromeClient(new WebChromeClient(){
            @Override public boolean onShowFileChooser(WebView view,ValueCallback<Uri[]> callback,FileChooserParams params){
                if(fileCallback!=null)fileCallback.onReceiveValue(null);fileCallback=callback;
                Intent intent=new Intent(Intent.ACTION_OPEN_DOCUMENT).setType("*/*").addCategory(Intent.CATEGORY_OPENABLE);
                intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE,params.getMode()==FileChooserParams.MODE_OPEN_MULTIPLE);
                startActivityForResult(intent,10);return true;
            }
            @Override public boolean onJsAlert(WebView view,String url,String text,JsResult result){message(text);result.confirm();return true;}
            @Override public boolean onJsConfirm(WebView view,String url,String text,JsResult result){new AlertDialog.Builder(MainActivity.this).setMessage(text).setPositiveButton("Confirm",(d,w)->result.confirm()).setNegativeButton("Cancel",(d,w)->result.cancel()).setOnCancelListener(d->result.cancel()).show();return true;}
        });
        web.setDownloadListener((url,agent,disposition,type,length)->{
            if(!local(Uri.parse(url))){message("Only local GP-SDR exports can be saved here.");return;}
            download=url;Intent save=new Intent(Intent.ACTION_CREATE_DOCUMENT).setType(type==null?"application/octet-stream":type).addCategory(Intent.CATEGORY_OPENABLE);
            save.putExtra(Intent.EXTRA_TITLE,URLUtil.guessFileName(url,disposition,type));startActivityForResult(save,11);
        });
        root.addView(web,new LinearLayout.LayoutParams(-1,0,1));setContentView(root);
        if(Build.VERSION.SDK_INT>=33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS)!=android.content.pm.PackageManager.PERMISSION_GRANTED)requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS},20);
        state.setText("Android preview • P25 local decoder not yet available. Choose storage, then Start.");
    }
    private Button button(String text,Runnable action){Button b=new Button(this);b.setText(text);b.setTextSize(12);b.setAllCaps(false);b.setOnClickListener(v->action.run());return b;}
    private boolean local(Uri uri){return !origin.isEmpty() && "http".equals(uri.getScheme()) && "127.0.0.1".equals(uri.getHost()) && uri.getPort()==Uri.parse(origin).getPort();}
    private void runState(){Button run=findViewById(android.R.id.content).findViewWithTag("run");run.setText(started?"Stop":"Start");}
    private void start(){
        if(started)return;
        Intent intent=new Intent(this,ReceiverService.class);startForegroundService(intent);bound=bindService(intent,connection,Context.BIND_AUTO_CREATE);
        started=true;state.setText("Starting local receiver…");runState();
    }
    private void poll(){
        if(!started||service==null)return;
        if(!service.error().isEmpty()){state.setText(service.error());return;}
        String url=service.url();
        if(url.isEmpty()){main.postDelayed(this::poll,200);return;}
        origin=url;web.loadUrl(url);
        state.setText("Local engine • P25: not ported yet • choose USB or add an RTL-TCP receiver in Hardware");
    }
    private void stop(){
        main.removeCallbacksAndMessages(null);web.stopLoading();web.loadUrl("about:blank");origin="";
        if(bound){unbindService(connection);bound=false;}stopService(new Intent(this,ReceiverService.class));service=null;started=false;
        state.setText("Stopping receiver. Wait for its notification to disappear before changing storage.");
    }
    private void performance(){
        String[] modes={"Auto (device memory)","Eco • 4 FPS / 256 bins","Balanced • 10 FPS / 512 bins","Detail • 20 FPS / 1024 bins","Performance • 40 FPS / 4096 bins"};
        int selected=getSharedPreferences("mobile",0).getInt("performance",0);
        new AlertDialog.Builder(this).setTitle("Display performance").setSingleChoiceItems(modes,selected,(dialog,which)->{
            getSharedPreferences("mobile",0).edit().putInt("performance",which).apply();applyPerformance();dialog.dismiss();
        }).setMessage("Changes graph rendering only. Receiver sample rate and Mapper concurrency stay under your control.").setNegativeButton("Close",null).show();
    }
    private void applyPerformance(){
        int mode=getSharedPreferences("mobile",0).getInt("performance",0);
        if(mode==0){ActivityManager.MemoryInfo info=new ActivityManager.MemoryInfo();getSystemService(ActivityManager.class).getMemoryInfo(info);mode=info.totalMem<4L*1024*1024*1024?1:2;}
        int fps=mode==1?4:mode==2?10:mode==3?20:40,bins=mode==1?256:mode==2?512:mode==3?1024:4096;double quality=mode==1?.5:mode==2?.75:mode==3?1:1.5;
        web.evaluateJavascript("(()=>{if(typeof displayPrefs==='undefined')return;Object.assign(displayPrefs,{fps:"+fps+",detail:"+bins+",quality:"+quality+"});localStorage.setItem('gpsdr-display-v2',JSON.stringify(displayPrefs));for(const [id,key] of [['display-fps','fps'],['display-detail','detail'],['display-quality','quality']]){let e=document.getElementById(id);if(e)e.value=displayPrefs[key];}})()",null);
    }
    private void applyMobileCapabilities(){
        web.evaluateJavascript("(()=>{document.documentElement.dataset.mobile='android';for(const view of ['decoders','transmit'])document.querySelector('[data-view=\"'+view+'\"]')?.remove();for(const id of ['live-mode','tuner-mode','mapper-mode']){let e=document.getElementById(id);if(!e)continue;for(const option of [...e.options])if(!['auto','am','nfm','wfm','usb','lsb','raw'].includes(option.value))option.remove();}let decoder=document.getElementById('mapper-decoder');if(decoder){decoder.value='auto';decoder.closest('label')?.remove();}let dialog=document.getElementById('missing-components-dialog');if(dialog?.open)dialog.close();dialog?.remove();})()",null);
    }
    private void storage(){
        if(started){message("Stop the receiver before changing storage. Existing recordings will not be deleted or moved.");return;}
        List<File> cards=Storage.cards(this);String[] labels=new String[cards.size()+1];labels[0]="Internal storage";
        for(int i=0;i<cards.size();i++)labels[i+1]="SD card • "+(cards.get(i).getUsableSpace()/1024/1024/1024)+" GB free";
        String selected=getSharedPreferences("mobile",0).getString("card","");int choice=0;
        for(int i=0;i<cards.size();i++)if(cards.get(i).getAbsolutePath().equals(selected))choice=i+1;
        new AlertDialog.Builder(this).setTitle("Recordings, IQ and exports").setSingleChoiceItems(labels,choice,(dialog,which)->{
            try{Storage.select(this,which==0?"":cards.get(which-1).getAbsolutePath());Storage.prepare(this);state.setText("Storage saved • "+labels[which]);dialog.dismiss();}
            catch(Exception e){message(e.getMessage());}
        }).setMessage("Settings and Mapper results remain internal. SD data is app-owned and is removed on uninstall. Export valuable captures first. Removing the card stops successful recording; there is no silent fallback.").setNegativeButton("Close",null).show();
    }
    private void message(String text){new AlertDialog.Builder(this).setMessage(text).setPositiveButton("OK",null).show();}
    @Override protected void onActivityResult(int request,int result,Intent data){
        super.onActivityResult(request,result,data);
        if(request==10 && fileCallback!=null){fileCallback.onReceiveValue(WebChromeClient.FileChooserParams.parseResult(result,data));fileCallback=null;}
        if(request==11 && result==RESULT_OK && data!=null && data.getData()!=null){
            Uri target=data.getData();String source=download,token=Uri.parse(origin).getQueryParameter("token");
            io.execute(()->{try{
                HttpURLConnection connection=(HttpURLConnection)new URL(source).openConnection();connection.setConnectTimeout(10000);connection.setReadTimeout(30000);connection.setInstanceFollowRedirects(false);connection.setRequestProperty("X-GP-SDR-Token",token);
                try{if(connection.getResponseCode()!=200)throw new IOException("Export failed: HTTP "+connection.getResponseCode());
                    try(InputStream in=connection.getInputStream();OutputStream out=getContentResolver().openOutputStream(target,"wt")){if(out==null)throw new IOException("Cannot write selected document");byte[] buffer=new byte[65536];int n;while((n=in.read(buffer))!=-1)out.write(buffer,0,n);}}
                finally{connection.disconnect();}
                main.post(()->message("Export saved."));
            }catch(Exception e){main.post(()->message("Export failed: "+e.getMessage()));}});
        }
    }
    @Override public void onBackPressed(){if(web.canGoBack())web.goBack();else new AlertDialog.Builder(this).setMessage("Leave the receiver running in the background?").setPositiveButton("Keep running",(d,w)->finish()).setNegativeButton("Stop receiver",(d,w)->{stop();finish();}).show();}
    @Override protected void onDestroy(){main.removeCallbacksAndMessages(null);if(bound)unbindService(connection);web.destroy();io.shutdown();super.onDestroy();}
}
