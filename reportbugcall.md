# Call System Bug Report

## Date: 2026-03-30

## Environment
- **Backend**: Go 1.25 + Chi v5 + LiveKit Go SDK on localhost:8083
- **Web**: Next.js 16 + livekit-client 2.18.0 on localhost:3001
- **Mobile**: Flutter 3.16 + livekit_client 2.3.0 on Xiaomi 23030RAC7Y (Android 15)
- **LiveKit Cloud**: wss://goservicemarketplace-68nmqloi.livekit.cloud
- **Redis**: Docker localhost:6380
- **PostgreSQL**: Docker localhost:5435

---

## Bug 1: Mobile camera permission always denied (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: `CAMERA` permission was missing from `AndroidManifest.xml`. Only `RECORD_AUDIO` was declared. `Permission.camera.request()` always returned `denied`.

**Fix applied**: Added `<uses-permission android:name="android.permission.CAMERA"/>` to AndroidManifest.xml.

**Remaining concern**: If the user previously denied the camera permission popup, Android caches the denial. The user must manually enable it in Settings → Apps → Marketplace → Permissions → Camera.

**Log evidence**:
```
[Call] acceptCall: callType=CallType.video
[Call] Camera permission denied, accepting as audio
[Call] _connectToRoom: callType=CallType.audio, isVideo=false
```

---

## Bug 2: No audio signal between mobile app and web (local)

**Status**: UNDER INVESTIGATION

**Symptoms**:
- Audio calls between web (PC) and mobile app (phone) produce no sound in either direction
- Audio calls between web-web (prod) work perfectly
- Audio calls between web-web (local, same PC) work perfectly

**Possible causes** (ordered by likelihood):

1. **PC volume/mute** — The user's PC sound was off/muted in 3 previous incidents. Must be checked FIRST before any code investigation.

2. **LiveKit room connection failure on mobile** — The log showed `LIVEKIT_URL not set — skipping room connection` when the APK was installed via `adb install` instead of `flutter run` with `--dart-define`. The `LIVEKIT_URL` is a compile-time constant (`String.fromEnvironment`). If the APK is built without `--dart-define=LIVEKIT_URL=...`, the mobile cannot connect to LiveKit.
   - **Verification**: Check if `[Call] Connecting to LiveKit room:` appears in logs. If `[Call] LIVEKIT_URL not set` appears instead, the APK was built without the define.

3. **Network/firewall** — The mobile is on WiFi (192.168.1.247), the PC is on the same network (192.168.1.156). Both connect to LiveKit Cloud (external). If the router blocks WebRTC/TURN traffic from the mobile, the peer connection would fail silently.

4. **Audio routing on mobile** — LiveKit on Android may route audio to the earpiece instead of the speaker. The `FlutterWebRTCPlugin: audioFocusChangeListener [Earpiece]` log suggests audio goes to the earpiece. The user might not hear it unless they hold the phone to their ear.

**Required diagnostics**:
- Check PC volume first
- Check mobile logs for `[Call] Connecting to LiveKit room:` vs `LIVEKIT_URL not set`
- Check if audio comes from earpiece (try speakerphone)
- Test with headphones/earbuds on mobile

---

## Bug 3: Video only one-way (PC→mobile, not mobile→PC) in local

**Status**: RESOLVED (likely permission issue)

**Root cause**: Same as Bug 1 — camera permission denied on mobile causes fallback to audio-only. The mobile publishes only audio tracks, not video. The PC publishes both audio + video.

**Result**: 
- PC→mobile: video shows (PC publishes video, mobile receives it) ✓
- Mobile→PC: no video (mobile doesn't publish video due to camera denial) ✗

**Fix**: Grant camera permission manually on the phone.

---

## Bug 4: Web video autoplay blocked (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: `<video muted={mirror}>` made the remote video element unmuted (`mirror=false`). Browsers block autoplay on unmuted `<video>` elements.

**Fix applied**: Changed to `<video muted>` (always muted). Audio goes through separate `<audio>` elements.

---

## Bug 5: Video tracks not rendered — race condition (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: `useVideoTracks` hook only listened for future LiveKit events. It never scanned for tracks already on the Room when the effect ran. Due to React's async useEffect timing, tracks published during connection were missed.

**Fix applied**: 
- Added scan for existing tracks after subscribing to events
- Added immediate `callType` state in `useCall` to avoid `"audio"` fallback
- Pre-loaded CallOverlay chunk on mount

---

## Bug 6: Mobile call_screen room listener race (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: `_listenForRoomEvents()` was called in `initState()` when `room` was still null (connects asynchronously). Track events were never caught.

**Fix applied**: `_listenForRoomChanges()` watches `callProvider` state and sets up the listener when room becomes available.

---

## Bug 7: Redis call TTL too short (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: TTL was reduced to 2 minutes during debugging. Calls longer than 2 minutes lost their Redis state, causing `call_ended` events to not be sent to the other party.

**Fix applied**: TTL increased to 30 minutes.

---

## Bug 8: Timer flicker between calls (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: 
- `call_accepted` handler had no state guard — stale events from previous calls started duplicate timers
- `call_ended` handler didn't validate call_id — late events from old calls killed current call
- `startDurationTimer` didn't clear existing interval

**Fix applied**: State guards + call_id validation + defensive clear.

---

## Bug 9: ref-after-dispose crash in mobile call_screen (RESOLVED)

**Status**: RESOLVED (2026-03-30)

**Root cause**: `ref.listenManual` callback in `_listenForRoomChanges` continued to fire after widget dispose. `ref.read()` crashed because the widget was already defunct.

**Fix applied**: Added `if (!mounted) return;` guard in the listener callback.

---

## Cleanup TODO

- [ ] Remove 23+ `console.log` debug statements from web call files
- [ ] Remove 15+ `print()` debug statements from mobile call files  
- [ ] Add backend state guard on `Accept()` method
- [ ] Consider audio routing (earpiece vs speaker) on mobile for calls

---

## Test Checklist

### Audio Calls
- [ ] Web → Web (local): audio both directions
- [ ] Web → Web (prod): audio both directions
- [ ] Web → Mobile app (local): audio both directions
- [ ] Mobile app → Web (local): audio both directions
- [ ] Web → Mobile browser (prod): audio both directions

### Video Calls
- [ ] Web → Web (local): video + audio both directions (requires v4l2loopback or 2 cameras)
- [ ] Web → Web (prod): video + audio both directions
- [ ] Web → Mobile app (local): video + audio both directions
- [ ] Mobile app → Web (local): video + audio both directions (requires camera permission)
- [ ] Web → Mobile browser (prod): video + audio both directions

### Call Lifecycle
- [ ] Initiate → Accept → Hangup (initiator): cleanup on both sides
- [ ] Initiate → Accept → Hangup (recipient): cleanup on both sides
- [ ] Initiate → Decline: proper cleanup
- [ ] Initiate → Timeout (30s): auto-decline
- [ ] Initiate → Accept → Navigate away (mobile): mini bar shows
- [ ] Initiate → Accept → Navigate pages (web): PiP persists
- [ ] Multiple rapid calls: no ghost calls or timer flicker

### Edge Cases
- [ ] Call while already in a call: blocked by `user_busy`
- [ ] Ghost call cleanup: unblocks after 30 min TTL
- [ ] Camera denied on video call: graceful fallback to audio
- [ ] LIVEKIT_URL not set: graceful skip with warning
- [ ] Network disconnect during call: UI stays visible for manual hangup
