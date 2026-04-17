document.addEventListener('DOMContentLoaded', () => {
    // Initialize Lucide icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }

    // Drawer Logic (Docs only)
    const menuToggle = document.getElementById("menu-toggle");
    const closeMenu = document.getElementById("close-menu");
    const mobileDrawer = document.getElementById("mobile-drawer");
    const mobileLinks = document.querySelectorAll(".mobile-link");

    if (menuToggle && closeMenu && mobileDrawer) {
        const toggleDrawer = () => mobileDrawer.classList.toggle("open");
        menuToggle.addEventListener("click", toggleDrawer);
        closeMenu.addEventListener("click", toggleDrawer);

        mobileLinks.forEach((link) => {
            link.addEventListener("click", () =>
                mobileDrawer.classList.remove("open")
            );
        });
    }

    // Scrollspy (Docs only)
    const sidebarLinks = document.querySelectorAll(".sidebar-link");
    const sections = document.querySelectorAll("section");

    if (sidebarLinks.length > 0 && sections.length > 0) {
        window.addEventListener("scroll", () => {
            let current = "";
            sections.forEach((s) => {
                const rect = s.getBoundingClientRect();
                if (window.scrollY >= s.offsetTop - 200) {
                    current = s.getAttribute("id");
                }
            });

            [...sidebarLinks, ...mobileLinks].forEach((l) => {
                l.classList.remove("active");
                if (l.getAttribute("href") === `#${current}`)
                    l.classList.add("active");
            });
        });
    }

    // Copy functionality
    document.querySelectorAll(".copy-button").forEach((button) => {
        button.addEventListener("click", async () => {
            const container =
                button.closest(".relative") ||
                button.closest(".group") ||
                button.parentElement;
            const code = container.querySelector("code");
            if (!code) return;

            const text = code.innerText.trim();

            try {
                await navigator.clipboard.writeText(text);

                const icon = button.querySelector("i");
                if (icon) {
                    icon.setAttribute("data-lucide", "check");
                    icon.classList.add("text-brand");
                    if (typeof lucide !== 'undefined') {
                        lucide.createIcons();
                    }

                    setTimeout(() => {
                        icon.setAttribute("data-lucide", "copy");
                        icon.classList.remove("text-brand");
                        if (typeof lucide !== 'undefined') {
                            lucide.createIcons();
                        }
                    }, 2000);
                }
            } catch (err) {
                console.error("Failed to copy: ", err);
            }
        });
    });
});
