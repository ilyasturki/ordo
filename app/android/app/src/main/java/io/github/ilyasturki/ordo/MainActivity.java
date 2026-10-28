package io.github.ilyasturki.ordo;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {

    @Override
    public void onCreate(Bundle savedInstanceState) {
        // Before super: the bridge reads the registered plugins while it builds the WebView.
        registerPlugin(FeedbackPlugin.class);
        super.onCreate(savedInstanceState);
    }
}
