# ML Kit's code scanner finds its registrars by reflection: R8's full mode would drop their constructors.
-keep class * implements com.google.firebase.components.ComponentRegistrar { <init>(); }
