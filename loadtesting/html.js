import http from "k6/http";
import { Counter } from "k6/metrics";
import { check } from "k6";

let indexFile = open("../test/testdata/html/index.html", "b"),
    styleFile = open("../test/testdata/html/style.css", "b"),
    headerFile = open("../test/testdata/html/header.html", "b"),
    footerFile = open("../test/testdata/html/footer.html", "b"),
    fontFile = open("../test/testdata/html/font.woff", "b"),
    imgFile = open("../test/testdata/html/img.gif", "b");

let failCounter = new Counter("failed requests");

export let options = {
  stages: [
    { duration: "10m", target: __ENV.MAX_VUS }
  ],
  thresholds: {
    "failed requests": [{
      threshold: "count<1", 
      abortOnFail: true,
    }]
  }
}

export default function() {
  let data = {
    "index.html": http.file(indexFile, "index.html"),
    "style.css": http.file(styleFile, "style.css"),
    "header.html": http.file(headerFile, "header.html"),
    "footer.html": http.file(footerFile, "footer.html"),
    "font.woff": http.file(fontFile, "font.woff"),
    "img.gif": http.file(imgFile, "img.gif")
  }
  let res = http.post(__ENV.BASE_URL + '/convert/html', data);
  check(res, {
    "is status 200": (r) => r.status === 200,
    "is not status 504": (r) => r.status !== 504,
    "is not status 500": (r) => r.status !== 500
  });
  if (res.status !== 200) {
    failCounter.add(1);
  }
}