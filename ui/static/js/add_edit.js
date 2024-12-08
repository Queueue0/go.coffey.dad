let tag_id = 0;
const add_tag = () => {
  const tagBox = document.getElementById("tagBox");
  let tagName = tagBox.value;
  if (!tagName) {
    return
  }

  
}

// Shamelessly stolen from StackOverflow
const string_to_color = (str) => {
  let hash = 0;
  str.split('').forEach(char => {
    hash = char.charCodeAt(0) + ((hash << 5) - hash);
  })

  let color = "#";
  for (let i = 0; i < 3; i++) {
    const value = (hash >> (i * 8)) & 0xff;
    color += value.toString(16).padStart(2, '0');
  }

  return color
}

const update_preview = () => {
  const bodyBox = document.getElementById("bodybox");
  const previewBox = document.getElementById("preview");
  const converter = new showdown.Converter();

  previewBox.innerHTML = converter.makeHtml(bodyBox.value);
}

document.addEventListener("DOMContentLoaded", () => {
  update_preview();
  document.getElementById("bodybox").addEventListener("input", update_preview);
  document.getElementById("insert-image").addEventListener("click", () => {window.open('/upload/choose-image')});
  document.getElementById("asDraftButton").addEventListener("click", () => {
    const form = document.getElementById("addEditForm");
    const asDraft = document.getElementById("asDraft");
    asDraft.value = "true";
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
