function getLimit() {
  return fetch("/limit").then(updateUI);
}

var last = null;

async function updateUI(resp) {
  let rdata = await resp.json();
  console.log(rdata);
  document.getElementById("limit").innerHTML = rdata.pull_limit;
  document.getElementById("remaining").innerHTML = rdata.pull_remaining;
  document.getElementById("checked_at").innerHTML = rdata.checked_at;
  document.getElementById("address").innerHTML = rdata.ip_address;
  last = rdata;
}
