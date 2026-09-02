package org.gpsdr.android;

import android.content.Context;
import android.os.Environment;
import android.system.Os;
import java.io.*;
import java.nio.file.Files;
import java.util.*;

/** Keep settings on internal storage; redirect only bulk data to app-owned SD storage. */
final class Storage {
    static final String[] BULK = {"IQ", "Recordings", "Exports"};
    static File root(Context c) { return new File(c.getFilesDir(), "GP-SDR"); }
    static List<File> cards(Context c) {
        List<File> cards = new ArrayList<>();
        for (File dir : c.getExternalFilesDirs(null))
            if (dir != null && Environment.isExternalStorageRemovable(dir) && Environment.MEDIA_MOUNTED.equals(Environment.getExternalStorageState(dir))) cards.add(dir);
        return cards;
    }
    static File prepare(Context c) throws Exception {
        File root = root(c);
        if (!root.isDirectory() && !root.mkdirs()) throw new IOException("Cannot create internal data folder");
        String selected = c.getSharedPreferences("mobile",0).getString("card", "");
        if (selected.isEmpty() && !c.getSharedPreferences("mobile",0).contains("card")) {
            List<File> available = cards(c);
            if (!available.isEmpty()) {
                selected = available.get(0).getAbsolutePath();
                c.getSharedPreferences("mobile",0).edit().putString("card", selected).commit();
            }
        }
        if (selected.isEmpty()) return root;
        File card = null;
        for (File candidate : cards(c)) if (candidate.getAbsolutePath().equals(selected)) card = candidate;
        if (card == null) throw new IOException("Selected SD card is not mounted. Insert it, or select internal storage. No fallback recording was started.");
        File targetRoot = new File(card, "GP-SDR");
        if (!targetRoot.isDirectory() && !targetRoot.mkdirs()) throw new IOException("SD card is not writable");
        for (String name : BULK) {
            File source = new File(root,name), target = new File(targetRoot,name);
            if (!target.isDirectory() && !target.mkdirs()) throw new IOException("Cannot create SD recording folder");
            boolean link = Files.isSymbolicLink(source.toPath());
            if (link && source.getCanonicalFile().equals(target.getCanonicalFile())) continue;
            if (source.exists() || link) {
                if (!link && source.isDirectory() && Objects.requireNonNull(source.list()).length > 0)
                    throw new IOException("Existing " + name + " data remains on internal storage. Export or move it before changing recording storage; nothing was deleted.");
                Files.deleteIfExists(source.toPath());
            }
            Os.symlink(target.getAbsolutePath(), source.getAbsolutePath());
        }
        return root;
    }
    static void select(Context c, String path) throws Exception {
        // Called only while receiver service is stopped. Never migrate or delete captures.
        if (path.isEmpty()) {
            for (String name:BULK) {
                File source=new File(root(c),name);
                if(Files.isSymbolicLink(source.toPath())) Files.deleteIfExists(source.toPath());
            }
        }
        c.getSharedPreferences("mobile",0).edit().putString("card",path).commit();
    }
}
