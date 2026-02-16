package org.devMandali.magnetPlay.util;

public class ByteUtil {
    public static String formatSize(long bytes) {
        String[] units = {"B", "KB", "MB", "GB", "TB", "PB"};
        if (bytes <= 0) return "0 B";

        // Calculate which index of 'units' to use
        int digitGroups = (int) (Math.log10(bytes) / Math.log10(1024));
        digitGroups = Math.min(digitGroups, units.length - 1);

        // Return formatted string: value / 1024^index
        return String.format("%.2f %s", bytes / Math.pow(1024, digitGroups), units[digitGroups]);
    }
}
