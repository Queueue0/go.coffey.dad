function closeSelf(sender) {
    try {
        window.opener.focus();
        if (window.location.pathname == "/choose-image") {
            window.opener.handleImageResult(sender.getAttribute("src"));
        }

        if (window.location.pathname == "/choose-header-image") {
            window.opener.handleHeaderImageResult(sender.getAttribute("src"));
        }
    }
    catch (err) {}
    window.close();
}

const images = document.querySelectorAll("img");
images.forEach((image) => image.addEventListener("click", () => closeSelf(image)));
