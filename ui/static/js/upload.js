document.addEventListener("DOMContentLoaded", () => {
  const fileInput = document.querySelector('input[type=file]');
    fileInput.onchange = () => {
      if (fileInput.files.length > 0) {
        const fileName = document.getElementById("filename");

        let files = ""
        for (const file of fileInput.files) {
          files += file.name + ";"
        }
        fileName.textContent = files;
      }
    };
});

