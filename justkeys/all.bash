#!/usr/bin/env bash
# Copyright 2014 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Script to build and launch the app on an android device.

set -e

./make.bash
adb install -r -s bin/JustKeys-debug.apk
adb shell am start -a android.intent.action.MAIN -n name.gordonklaus.justkeys/android.app.NativeActivity
