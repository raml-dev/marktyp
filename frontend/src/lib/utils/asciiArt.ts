type Pixel = {
  x: number;
  y: number;
};

const glyphs: Record<string, string[]> = {
  M: ['10001', '11011', '10101', '10101', '10001', '10001', '10001'],
  A: ['01110', '10001', '10001', '11111', '10001', '10001', '10001'],
  R: ['11110', '10001', '10001', '11110', '10100', '10010', '10001'],
  K: ['10001', '10010', '10100', '11000', '10100', '10010', '10001'],
  T: ['11111', '00100', '00100', '00100', '00100', '00100', '00100'],
  Y: ['10001', '10001', '01010', '00100', '00100', '00100', '00100'],
  P: ['11110', '10001', '10001', '11110', '10000', '10000', '10000'],
};

const logo = ['1111110', '1000010', '1011000', '1001100', '1000110', '1000010', '1111110'];

function pixels(rows: string[], xOffset = 0): Pixel[] {
  return rows.flatMap((row, y) =>
    [...row].flatMap((value, x) => (value === '1' ? [{ x: x + xOffset, y }] : [])),
  );
}

export const marktypWordmarkPixels = [...'MARKTYP'].flatMap((letter, index) =>
  pixels(glyphs[letter], index * 6),
);
export const marktypLogoPixels = pixels(logo);
