const add_tag = () => {
  const tagBox = document.getElementById("tagBox");
  let tagName = tagBox.value;
  if (!tagName) {
    return
  }

  let color = string_to_color(tagName);
  let textColor = get_text_color(color);

  const control = document.createElement("div");
  control.classList.add("control");

  const tags = document.createElement("div");
  tags.classList.add("tags");
  tags.classList.add("has-addons");

  control.appendChild(tags);

  const tag = document.createElement("span");
  tag.classList.add("tag");
  tag.style.setProperty("background-color", color);
  tag.style.setProperty("color", textColor);
  tag.innerHTML = tagName;

  const del = document.createElement("button");
  del.type = "button";
  del.classList.add("tag");
  del.classList.add("is-delete");
  del.addEventListener("click", (e) => {
    e.target.parentNode.parentNode.remove();
  });

  tags.appendChild(tag);
  tags.appendChild(del);

  document.getElementById("tagZone").appendChild(control);
  tagBox.value = "";
}

const touch_tags = () => {
  const tags = document.querySelectorAll("span.tag");

  tags.forEach((tag) => {
    let color = tag.className.split(' ')[1];
    let textColor = get_text_color(color);
    tag.style.setProperty("background-color", color);
    tag.style.setProperty("color", textColor);
  });

  const delBtns = document.querySelectorAll("button.tag.is-delete");
  delBtns.forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.target.parentNode.parentNode.remove();
    });
  });
}

// Shamelessly stolen from StackOverflow
const string_to_color = (str) => {
  let hash = 0;
  str.split('').forEach(char => {
    hash = char.charCodeAt(0) + ((hash << 5) - hash);
  });

  let color = "#";
  for (let i = 0; i < 3; i++) {
    const value = (hash >> (i * 8)) & 0xff;
    color += value.toString(16).padStart(2, '0');
  }

  return color;
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

const add_tag_fields = () => {
  // This is fragile, I'll change it if it becomes a problem
  const tags = document.querySelectorAll("span.tag");
  
  const form = document.getElementById("addEditForm");

  tags.forEach((tag, i) => {
    let name = tag.innerHTML;
    let color = tag.style.getPropertyValue("background-color");
    color = color.split("(")[1].split(")")[0];
    parts = color.split(", ");
    parts = parts.map((v) => {
      v = parseInt(v).toString(16);
      return (v.length==1) ? "0"+v : v;
    });
    color = "#"+parts.join("");

    const nameInput = document.createElement("input");
    nameInput.type = "hidden";
    nameInput.value = name;
    nameInput.name = "Tags[" + i + "].Name";

    const colorInput = document.createElement("input");
    colorInput.type = "hidden";
    colorInput.value = color;
    colorInput.name = "Tags[" + i + "].Color";

    form.appendChild(nameInput);
    form.appendChild(colorInput);
  });
}

const update_preview = () => {
  const bodyBox = document.getElementById("bodybox");
  const previewBox = document.getElementById("preview");
  const converter = new showdown.Converter();

  previewBox.innerHTML = converter.makeHtml(bodyBox.value);
}

document.addEventListener("DOMContentLoaded", () => {
  update_preview();
  touch_tags();
  document.getElementById("bodybox").addEventListener("input", update_preview);
  document.getElementById("insert-image").addEventListener("click", () => {window.open('/upload/choose-image')});
  document.getElementById("asDraftButton").addEventListener("click", () => {
    add_tag_fields();

    const form = document.getElementById("addEditForm");
    const asDraft = document.getElementById("asDraft");
    asDraft.value = "true";
    form.submit();
  });

  document.getElementById("submitBtn").addEventListener("click", () => {
    add_tag_fields();

    const form = document.getElementById("addEditForm");
    form.submit();
  });

  document.getElementById("useTitleBtn").addEventListener("click", () => {
    const urlBox = document.getElementById("url");
    const titleBox = document.getElementById("title");
    let url = titleBox.value;
    url = url.toLowerCase()
    url = url.replaceAll(/[^0-9a-z ]/gi, '');
    url = url.trim();
    url = url.replaceAll(" ", "-");

    urlBox.value = url;
  });

  document.getElementById("addTagBtn").addEventListener("click", add_tag);
});

function handlePopupResult(result) {
  const imageStr = '![](' + result + ')';
  const bodybox = document.getElementById("bodybox");
  const cursorPos = bodybox.selectionStart;
  const end = bodybox.selectionEnd;
  const bbValue = bodybox.value;
  bodybox.value = bbValue.slice(0, cursorPos) + imageStr + bbValue.slice(cursorPos);
  bodybox.focus();
  bodybox.selectionEnd = end + 2;
  update_preview();
}
