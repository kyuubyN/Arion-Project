#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-only

import os
import sys
import urllib.parse
import urllib.request

try:
    import gi
    gi.require_version("Gtk", "3.0")
    try:
        gi.require_version("WebKit2", "4.1")
    except ValueError:
        gi.require_version("WebKit2", "4.0")
    from gi.repository import Gtk, WebKit2, GLib
except Exception as error:
    print(f"Arion requires GTK 3 and WebKit2 for this transitional shell: {error}")
    sys.exit(1)

GLib.set_prgname("arion")
GLib.set_application_name("Arion")


class ArionWindow(Gtk.Window):
    def __init__(self, url):
        super().__init__(title="Arion — Galeria de mídia")
        self.url = url
        self.set_wmclass("arion", "Arion")
        self.set_default_size(1280, 800)
        self.set_position(Gtk.WindowPosition.CENTER)

        icon_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "assets", "icon.png")
        if os.path.exists(icon_path):
            try:
                self.set_icon_from_file(icon_path)
                Gtk.Window.set_default_icon_from_file(icon_path)
            except Exception:
                pass

        gtk_settings = Gtk.Settings.get_default()
        gtk_settings.set_property("gtk-application-prefer-dark-theme", True)

        self.webview = WebKit2.WebView()
        web_settings = self.webview.get_settings()
        web_settings.set_enable_developer_extras(False)
        web_settings.set_enable_media_stream(True)
        web_settings.set_enable_webrtc(False)
        self.webview.load_uri(url)
        self.add(self.webview)
        self.connect("destroy", self.on_destroy)

    def on_destroy(self, _widget):
        try:
            parsed = urllib.parse.urlparse(self.url)
            token = urllib.parse.parse_qs(parsed.fragment).get("session", [""])[0]
            endpoint = f"{parsed.scheme}://{parsed.netloc}/api/shutdown"
            request = urllib.request.Request(endpoint, method="POST", headers={"Authorization": f"Bearer {token}"})
            urllib.request.urlopen(request, timeout=1)
        except Exception:
            pass
        Gtk.main_quit()


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: arion-window.py <local-arion-url>")
        sys.exit(2)
    window = ArionWindow(sys.argv[1])
    window.show_all()
    Gtk.main()
