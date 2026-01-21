const videos = document.querySelectorAll("video");
let current = 0;
const DISPLAY_TIME = 8000;   // ms per video
const FADE_TIME = 500;      // must match CSS

function showVideo(index) {
  videos.forEach((v, i) => {
    v.classList.toggle("active", i === index);
  });
}

function nextVideo() {
  videos[current].classList.remove("active");
  current = (current + 1) % videos.length;
  showVideo(current);
}

showVideo(current);
setInterval(nextVideo, DISPLAY_TIME);
