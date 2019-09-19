import http from "k6/http";
import { check } from "k6";

let pdf1File = open("../test/testdata/pdf/gotenberg.pdf", "b"),
    pdf2File = open("../test/testdata/pdf/gotenberg_bis.pdf", "b");

export default function() {
  var data = {
      "gotenberg.pdf": http.file(pdf1File, "gotenberg.pdf"),
      "gotenberg_bis.pdf": http.file(pdf2File, "gotenberg_bis.pdf")
  }
  var res = http.post(__ENV.BASE_URL + '/merge', data);
  check(res, {
    "is status 200": (r) => r.status === 200,
    "is not status 504": (r) => r.status !== 504,
    "is not status 500": (r) => r.status !== 500,
  });
}