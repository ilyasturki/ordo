package io.github.ilyasturki.ordo;

import android.os.Build;
import android.view.HapticFeedbackConstants;
import android.view.SoundEffectConstants;
import android.view.View;

import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

/**
 * The tap sound and haptics a native control gives, played through the WebView so they follow
 * the system's touch sound and touch vibration settings.
 */
@CapacitorPlugin(name = "Feedback")
public class FeedbackPlugin extends Plugin {

    @PluginMethod
    public void click(PluginCall call) {
        View view = view();
        if (view != null) {
            getActivity().runOnUiThread(() -> view.playSoundEffect(SoundEffectConstants.CLICK));
        }
        call.resolve();
    }

    @PluginMethod
    public void haptic(PluginCall call) {
        View view = view();
        int effect = "hold".equals(call.getString("kind"))
            ? HapticFeedbackConstants.LONG_PRESS
            : Build.VERSION.SDK_INT >= Build.VERSION_CODES.R
                ? HapticFeedbackConstants.CONFIRM
                : HapticFeedbackConstants.VIRTUAL_KEY;
        if (view != null) {
            getActivity().runOnUiThread(() -> view.performHapticFeedback(effect));
        }
        call.resolve();
    }

    private View view() {
        return getBridge() == null ? null : getBridge().getWebView();
    }
}
