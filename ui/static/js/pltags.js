const touch_tags = () => {
  const tags = document.querySelectorAll("a.post-tag");

  tags.forEach((tag) => {
    let color = tag.className.split(' ')[1];
    let textColor = get_text_color(color);
    tag.style.setProperty("background-color", color);
    tag.style.setProperty("color", textColor);
  });
}

const get_text_color = (c) => {
  c = c.replace('#', '');
  let r = Number("0x"+c.substring(0, 2));
  let g = Number("0x"+c.substring(2, 4));
  let b = Number("0x"+c.substring(4));

  let convert = (v) => {
    v /= 255;

    v = v <= 0.03928 ? v/12.92 : ((v+0.055)/1.055)**2.4;

    return v;
  }

  r = convert(r);
  g = convert(g);
  b = convert(b);

  let L = 0.2126 * r + 0.7152 * g + 0.0722 * b;
  return L > 0.179 ? "#000000" : "#ffffff";
}

document.addEventListener("DOMContentLoaded", () => {
  touch_tags();
});
