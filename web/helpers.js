function parseGetParams() {
    return new URLSearchParams(window.location.search);
}

function parseStartEndParams(q) {
    let start;
    let end;

    if (q.get("start") === "lastweek") {
        start = moment().subtract(1, "weeks").startOf('isoWeek');
        if (!q.get("end")) {
            end = moment().subtract(1, "weeks").endOf('isoWeek');
        }
    }
    if (!start && q.get("start")) {
        start = moment(q.get("start"), "YYYY-MM-DD");
        if (!q.get("end")) {
            end = moment(start).add(1, "weeks");
        }
    }
    if (!end && q.get("end")) {
        end = moment(q.get("end"), "YYYY-MM-DD");
        if (!q.get("start")) {
            start = moment(end).subtract(1, "weeks");
        }
    }
    if (!start) { start = moment().subtract(0, 'weeks').startOf('isoWeek'); }
    if (!end) { end = moment().subtract(0, 'weeks').endOf('isoWeek'); }
    return [start, end];
}