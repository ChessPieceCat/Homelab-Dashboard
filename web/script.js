// Prevent overlapping refresh requests.
let refreshingContainers = false;
let refreshingPerformance = false;

// Prevent the dashboard from being replaced while a container action
// is being submitted.
let isSubmitting = false;


async function refreshContainers() {
    if (refreshingContainers || isSubmitting) {
        return;
    }

    refreshingContainers = true;

    try {
        const response = await fetch("/containers");

        if (!response.ok) {
            throw new Error(`Container refresh failed: ${response.status}`);
        }

        const html = await response.text();

        if (!isSubmitting) {
            document.querySelector(".containers").innerHTML = html;
        }
    } catch (error) {
        console.error("Failed to refresh containers:", error);
    } finally {
        refreshingContainers = false;
    }
}


async function refreshPerformance() {
    if (refreshingPerformance || isSubmitting) {
        return;
    }

    refreshingPerformance = true;

    try {
        const response = await fetch("/performance");

        if (!response.ok) {
            throw new Error(`Performance refresh failed: ${response.status}`);
        }

        const html = await response.text();

        if (!isSubmitting) {
            document.querySelector(".performance-metrics").innerHTML = html;
        }
    } catch (error) {
        console.error("Failed to refresh performance:", error);
    } finally {
        refreshingPerformance = false;
    }
}


document.addEventListener("submit", (event) => {
    if (event.target.closest(".containers form")) {
        isSubmitting = true;

        const buttons = event.target.querySelectorAll("button");

        buttons.forEach(button => {
            button.disabled = true;
        });
    }
});


async function refresh() {
    await Promise.all([
        refreshContainers(),
        refreshPerformance()
    ]);

    setTimeout(refresh, 1000);
}


refresh();
