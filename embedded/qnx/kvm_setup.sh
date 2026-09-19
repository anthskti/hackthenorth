#!/bin/sh
# KVM setup: GPIO UART for the Leonardo + USB capture card for the camera.
# Output is logged to /tmp/kvm_setup.log

{
  echo "kvm_setup: starting $(date)"

  # UART0 on GPIO14/15 -> /dev/ser1
  if [ ! -e /dev/ser1 ]; then
    msix-rp1 -m set 25l
    gpio-rp1 set 14,15 a4
    devc-serpl011-rpi5 -b115200 -c50000000 -e -F -u1 0x1f00030000,185
    waitfor /dev/ser1 5
  fi
  stty baud=115200 par=none bits=8 stopb=1 < /dev/ser1
  chmod 666 /dev/ser1

  # Restart the sensor service with the USB capture card config
  slay sensor 2>/dev/null
  sleep 1
  sensor -U 521:521,1001 -r /data/share/sensor -c /usr/etc/config/sensor/usb/usb_camera_jpeg.conf

  echo "kvm_setup: done $(date)"
} > /tmp/kvm_setup.log 2>&1
