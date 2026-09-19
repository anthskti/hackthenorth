/*
 * kvm_keyboard.ino - Arduino Leonardo as a USB keyboard + mouse over serial.
 *
 * Commands come from the Pi on Serial1 (pin 0 = RX), or from the laptop over
 * USB serial (/dev/ttyACM0) for testing. 115200 baud, one command per line.
 *
 * Wiring (Pi 5 -> Leonardo):
 *   Pi pin 8 (GPIO14 TXD) -> Leonardo pin 0 (RX)
 *   Pi pin 6 (GND)        -> Leonardo GND
 *   Do NOT connect Leonardo TX (5 V) to the Pi.
 *
 * KEYBOARD
 *   t<text>   type text              e.g. "thello world"
 *   p<hex>    press and hold a key   e.g. "p80" (Left Ctrl)
 *   r<hex>    release a key          e.g. "r80"
 *   k<hex>    tap a key              e.g. "kB0" (Enter)
 *   a         release all keys
 *
 * MOUSE
 *   m<XXXX><YYYY>        move to an ABSOLUTE position, 8 hex digits, each axis
 *                        0000-7FFF across the screen. "m3FFF3FFF" = centre.
 *   m<dx>,<dy>[,<wheel>] RELATIVE move, signed decimal. e.g. "m10,-5"
 *   b<L|R|M><d|u|c>      button down / up / click. e.g. "bLc" = left click
 *   w<n>                 scroll wheel, signed decimal. "w1" up, "w-1" down.
 *   z                    release all mouse buttons
 *
 * The two move spellings are told apart by the comma. The dashboard sends
 * absolute, because a click has to land where the operator aimed.
 *
 * Key codes (hex):
 *   80 L-Ctrl  81 L-Shift  82 L-Alt  83 L-GUI (Win/Super)
 *   B0 Enter   B1 Esc      B2 Backspace  B3 Tab  20 Space
 *   D7 Right   D8 Left     D9 Down   DA Up
 *   D1 Insert  D4 Delete   D2 Home   D5 End  D3 PgUp  D6 PgDn
 *   C2..CD = F1..F12
 *   Printable characters use their ASCII code, e.g. 61 = 'a'.
 *
 * Absolute mouse needs the HID-Project library (NicoHood), which declares a
 * pointer with an absolute HID descriptor. Stock Mouse.h is relative only and
 * cannot place the cursor at a known screen position, which is what a KVM
 * needs. Install: Library Manager -> "HID-Project".
 */
#include <HID-Project.h>

static char line[128];
static uint8_t len = 0;

static uint8_t parseHex(const char *s)
{
  return (uint8_t)strtoul(s, NULL, 16);
}

/* Read exactly n hex digits starting at s. Returns 0 on a short or bad field. */
static uint16_t parseHexN(const char *s, uint8_t n)
{
  uint16_t v = 0;
  for (uint8_t i = 0; i < n; i++) {
    char c = s[i];
    uint8_t d;
    if (c >= '0' && c <= '9')      d = c - '0';
    else if (c >= 'a' && c <= 'f') d = c - 'a' + 10;
    else if (c >= 'A' && c <= 'F') d = c - 'A' + 10;
    else return 0;
    v = (uint16_t)((v << 4) | d);
  }
  return v;
}

/* Which pointer the cursor was last moved with. Buttons follow it. */
static bool g_absolute = false;

static void pressButton(uint8_t b)
{
  if (g_absolute) AbsoluteMouse.press(b); else Mouse.press(b);
}

static void releaseButton(uint8_t b)
{
  if (g_absolute) AbsoluteMouse.release(b); else Mouse.release(b);
}

static int8_t clamp8(int v)
{
  if (v >  127) return  127;
  if (v < -127) return -127;
  return (int8_t)v;
}

/* 'L' | 'R' | 'M' -> the HID button constant */
static uint8_t buttonOf(char c)
{
  if (c == 'R') return MOUSE_RIGHT;
  if (c == 'M') return MOUSE_MIDDLE;
  return MOUSE_LEFT;
}

static void handleLine(char *cmd)
{
  if (cmd[0] == '\0') return;

  switch (cmd[0]) {
    case 't':                       // type text
      BootKeyboard.print(cmd + 1);
      break;
    case 'p':                       // press and hold
      BootKeyboard.press((KeyboardKeycode)parseHex(cmd + 1));
      break;
    case 'r':                       // release one key
      BootKeyboard.release((KeyboardKeycode)parseHex(cmd + 1));
      break;
    case 'k': {                     // tap
      KeyboardKeycode k = (KeyboardKeycode)parseHex(cmd + 1);
      BootKeyboard.press(k);
      delay(20);
      BootKeyboard.release(k);
      break;
    }
    case 'a':                       // release every key
      BootKeyboard.releaseAll();
      break;

    case 'm': {
      // Absolute "mXXXXYYYY" or relative "m<dx>,<dy>[,<wheel>]", told apart
      // by the comma -- same rule the QNX daemon uses to validate them.
      char *comma = strchr(cmd + 1, ',');
      if (comma == NULL) {
        if (strlen(cmd + 1) < 8) break;
        uint16_t x = parseHexN(cmd + 1, 4);
        uint16_t y = parseHexN(cmd + 5, 4);
        // AbsoluteMouse spans the screen over a signed 16-bit range, so map
        // our unsigned 0..0x7FFF onto -32768..32767.
        AbsoluteMouse.moveTo((int16_t)((int32_t)x * 2 - 32768),
                             (int16_t)((int32_t)y * 2 - 32768));
        g_absolute = true;
      } else {
        int dx = atoi(cmd + 1);
        int dy = atoi(comma + 1);
        char *second = strchr(comma + 1, ',');
        int wheel = second ? atoi(second + 1) : 0;
        Mouse.move(clamp8(dx), clamp8(dy), clamp8(wheel));
        g_absolute = false;
      }
      break;
    }
    case 'b': {                     // b<L|R|M><d|u|c>
      if (strlen(cmd) < 3) break;
      uint8_t b = buttonOf(cmd[1]);
      // Send from whichever pointer the cursor was last moved with, or the
      // click arrives from a device that is not where the user is pointing.
      if (cmd[2] == 'd')      pressButton(b);
      else if (cmd[2] == 'u') releaseButton(b);
      else if (cmd[2] == 'c') { pressButton(b); delay(20); releaseButton(b); }
      break;
    }
    case 'w':                       // scroll wheel, signed decimal
      // Always relative: AbsoluteMouse.move(0,0,..) would warp the pointer
      // to the top-left corner on its way to scrolling.
      Mouse.move(0, 0, clamp8(atoi(cmd + 1)));
      break;
    case 'z':                       // release all mouse buttons
      AbsoluteMouse.releaseAll();
      Mouse.release(MOUSE_LEFT);
      Mouse.release(MOUSE_RIGHT);
      Mouse.release(MOUSE_MIDDLE);
      break;

    default:
      break;                        // ignore unknown commands
  }
}

static void feed(char c)
{
  if (c == '\r') return;
  if (c == '\n') {
    line[len] = '\0';
    handleLine(line);
    len = 0;
  } else if (len < sizeof(line) - 1) {
    line[len++] = c;
  }
}

void setup()
{
  Serial.begin(115200);    // USB serial: testing from the laptop
  Serial1.begin(115200);   // hardware UART: commands from the Pi
  BootKeyboard.begin();    // boot protocol: works in BIOS, unlike stock Keyboard.h
  AbsoluteMouse.begin();
  Mouse.begin();          // relative moves, for the m<dx>,<dy> spelling
}

void loop()
{
  while (Serial1.available()) feed((char)Serial1.read());
  while (Serial.available())  feed((char)Serial.read());
}
