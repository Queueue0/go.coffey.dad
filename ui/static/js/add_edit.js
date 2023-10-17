update_preview = () => {
  const bodyBox = document.getElementById("bodybox");
  const previewBox = document.getElementById("preview");
  const converter = new showdown.Converter();

  previewBox.innerHTML = converter.makeHtml(bodyBox.value);
}

document.addEventListener("DOMContentLoaded", () => {
  update_preview();
  document.getElementById("bodybox").addEventListener("input", update_preview);
  document.getElementById("insert-image").addEventListener("click", () => {window.open('/upload/choose-image')});
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