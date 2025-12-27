#!/bin/sh

find . \
  \( -path './clawc/internal/imports/testing/config' -o -path '*claw_vendor*' \) -prune -o \
  -type f -name '*.claw' -exec clawc {} \;
