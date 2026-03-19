package com.musicd.player.alex1800;

import android.os.Bundle;
import android.webkit.WebSettings;
import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
	@Override
	protected void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);

		// Allow loading HTTP resources from an HTTPS page (required for
		// connecting to local/LAN backend servers over plain HTTP).
		WebSettings webSettings = getBridge().getWebView().getSettings();
		webSettings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
	}
}
