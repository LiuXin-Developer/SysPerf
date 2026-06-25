(function () {
    "use strict";

    var toggle = document.getElementById("navToggle");
    var links = document.getElementById("navLinks");

    if (toggle && links) {
        toggle.addEventListener("click", function () {
            links.classList.toggle("open");
        });

        links.querySelectorAll("a").forEach(function (a) {
            a.addEventListener("click", function () {
                links.classList.remove("open");
            });
        });
    }

    /* 滚动时导航栏增加阴影感 */
    var nav = document.querySelector(".nav");
    if (nav) {
        window.addEventListener("scroll", function () {
            nav.style.boxShadow = window.scrollY > 10
                ? "0 4px 24px rgba(0,0,0,0.3)"
                : "none";
        }, { passive: true });
    }

    /* 可选：在 index.html 前设置 window.REPO_URL = "https://github.com/owner/repo" */
    var repo = window.REPO_URL || "";
    if (repo) {
        var gh = document.getElementById("githubLink");
        var rd = document.getElementById("readmeLink");
        if (gh) gh.href = repo;
        if (rd) rd.href = repo + "/blob/main/README.md";
    }
})();
