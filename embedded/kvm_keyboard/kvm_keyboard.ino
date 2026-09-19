/*
 * kvm_keyboard.ino - Arduino Leonardo as a USB keyboard controlled over serial.
 *
 * Commands come from the Pi on Serial1 (pin 0 = RX), or from the laptop over
 * USB serial (/dev/ttyACM0) for testing. 115200 baud, one command per line.
 *
 * Wiring (Pi 5 -> Leonardo):
 *   Pi pin 8 (GPIO14 TXD) -> Leonardo pin 0 (RX)
 *   Pi pin 6 (GND)        -> Leonardo GND
 *   Do NOT connect Leonardo TX (5 V) to the Pi.
 *
 * Commands:
 *   t<text>   type text              e.g. "thello world"
 *   p<hex>    press and hold a key   e.g. "p80" (Left Ctrl)
 *   r<hex>    release a key          e.g. "r80"
 *   k<hex>    tap a key              e.g. "kB0" (Enter)
 *   a         release all keys
 *
 * Key codes (hex):
 *   80 L-Ctrl  81 L-Shift  82 L-Alt  83 L-GUI (Win/Super)
 *   B0 Enter   B1 Esc      B2 Backspace  B3 Tab  20 Space
 *   D7 Right   D8 Left     D9 Down   DA Up
 *   D1 Insert  D4 Delete   D2 Home   D5 End  D3 PgUp  D6 PgDn
 *   C2..CD = F1..F12
 *   Printable characters use their ASCII code, e.g. 61 = 'a'.
 */
#include <Keyboard.h>

static char line[128];
static uint8_t len = 0;

static uint8_t parseHex(const char *s)
{
  return (uint8_t)strtoul(s, NULL, 16);
}

static void handleLine(char *cmd)
{
  if (cmd[0] == '\0') return;

  switch (cmd[0]) {
    case 't':                       // type text
      Keyboard.print(cmd + 1);
      break;
    case 'p':                       // press and hold
      Keyboard.press(parseHex(cmd + 1));
      break;
    case 'r':                       // release one key
      Keyboard.release(parseHex(cmd + 1));
      break;
    case 'k': {                     // tap
      uint8_t k = parseHex(cmd + 1);
      Keyboard.press(k);
      delay(20);
      Keyboard.release(k);
      break;
    }
    case 'a':                       // release everything
      Keyboard.releaseAll();
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
  Keyboard.begin();
}

void loop()
{
  while (Serial1.available()) feed((char)Serial1.read());
  while (Serial.available())  feed((char)Serial.read());
}
