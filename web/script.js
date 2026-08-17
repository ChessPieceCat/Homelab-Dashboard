

async function refreshContainers() {

    const response = await fetch("/containers");
    const html = await response.text();

    document.querySelector(".containers").innerHTML = html;
}

async function refreshPerformance() {

    const response = await fetch("/performance");
    const html = await response.text();

    document.querySelector(".performance-metrics").innerHTML = html;
}

setInterval(() => {
    refreshContainers();
    refreshPerformance();
}, 1000);