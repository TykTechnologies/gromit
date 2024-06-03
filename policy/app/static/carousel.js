async function fetchAndRenderCarousel() {
    try {
        const response = await fetch('/variations');
        const data = await response.json();

        const carousel = document.getElementById('carousel');

        for (const [title, content] of Object.entries(data)) {
            const item = createCarouselItem(title, content.Leaves);
            carousel.appendChild(item);
        }
    } catch (error) {
        console.error('Error fetching data:', error);
    }
}

function createCarouselItem(title, content) {
    const item = document.createElement('div');
    item.className = 'carousel-item';

    const titleElement = document.createElement('h3');
    titleElement.textContent = title;
    item.appendChild(titleElement);

    const contentElement = document.createElement('pre');
    contentElement.textContent = JSON.stringify(content, null, 2);
    item.appendChild(contentElement);

    return item;
}

//fetchAndRenderCarousel();
