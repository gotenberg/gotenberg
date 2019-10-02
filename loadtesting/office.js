import http from "k6/http";
import { Counter } from "k6/metrics";
import { check } from "k6";

let documentFile = open("../test/testdata/office/document.docx", "b");

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
      "document.docx": http.file(documentFile, "document.docx")
  }
  let res = http.post(__ENV.BASE_URL + '/convert/office', data);
  check(res, {
    "is status 200": (r) => r.status === 200,
    "is not status 504": (r) => r.status !== 504,
    "is not status 500": (r) => r.status !== 500,
  });
  if (res.status !== 200) {
    failCounter.add(1);
  }
}