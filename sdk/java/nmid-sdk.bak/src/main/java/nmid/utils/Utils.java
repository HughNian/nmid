package nmid.utils;

import java.util.concurrent.atomic.AtomicLong;

public class Utils {
    private static final AtomicLong idGenerator = new AtomicLong(0);

    public static String GetId() {
        // return java.util.UUID.randomUUID().toString().replaceAll("-", "");
        long value = System.nanoTime() << 32;
        long next = idGenerator.incrementAndGet();
        return Long.toString(value + next);
    }
}
