function closeSelf(sender) {
    try {
        window.opener.focus();
        window.opener.handlePopupResult(sender.getAttribute("src"));
    }
    catch (err) {}
    window.close();
}

const images = document.querySelectorAll("img");
images.forEach((image) => image.addEventListener("click", () => closeSelf(image)));
