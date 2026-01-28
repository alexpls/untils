const EMPTY_SCHEDULE = {
  day0: false, // sunday
  day1: false, // monday
  day2: false, // tuesday
  day3: false, // wednesday
  day4: false, // thursday
  day5: false, // friday
  day6: false, // saturday

  hour0: false, // midnight
  hour1: false, // 1am
  hour2: false, // 2am
  hour3: false, // 3am
  hour4: false, // 4am
  hour5: false, // 5am
  hour6: false, // 6am
  hour7: false, // 7am
  hour8: false, // 8am
  hour9: false, // 9am
  hour10: false, // 10am
  hour11: false, // 11am
  hour12: false, // 12pm
  hour13: false, // 1pm
  hour14: false, // 2pm
  hour15: false, // 3pm
  hour16: false, // 4pm
  hour17: false, // 5pm
  hour18: false, // 6pm
  hour19: false, // 7pm
  hour20: false, // 8pm
  hour21: false, // 9pm
  hour22: false, // 10pm
  hour23: false, // 11pm
}

function empty() {
  return { ...EMPTY_SCHEDULE }
}

function everyMorning(schedule) {
  for (const key in schedule) {
    if (key.startsWith("day")) {
      schedule[key] = true
    } else {
      schedule[key] = false
    }
  }
  schedule.hour9 = true
}

function everyHour(schedule) {
  for (const key in schedule) {
    schedule[key] = true
  }
}

function everyWeek(schedule) {
  for (const key in schedule) {
    schedule[key] = false
  }
  schedule.day1 = true
  schedule.hour8 = true
}

// Merge consecutive numbers into ranges, e.g. [1,2,3,5,6] => "1-3,5-6"
function cronMergeConsecutive(nums) {
  if (nums.length === 0) return ""

  nums.sort((a, b) => a - b)

  const ranges = []
  let start = nums[0]
  let end = nums[0]

  for (let i = 1; i < nums.length; i++) {
    if (nums[i] === end + 1) {
      end = nums[i]
    } else {
      ranges.push(start === end ? `${start}` : `${start}-${end}`)
      start = nums[i]
      end = nums[i]
    }
  }

  ranges.push(start === end ? `${start}` : `${start}-${end}`)

  return ranges.join(",")
}

// Parse a cron field like "1-3,5,7-9" into an array of numbers [1,2,3,5,7,8,9]
function cronParseField(field) {
  if (field === "*") {
    return null // indicates "all"
  }

  const nums = []
  const parts = field.split(",")

  for (const part of parts) {
    if (part.includes("-")) {
      const [start, end] = part.split("-").map((n) => parseInt(n, 10))
      for (let i = start; i <= end; i++) {
        nums.push(i)
      }
    } else {
      nums.push(parseInt(part, 10))
    }
  }

  return nums
}

function fromCron(cron) {
  const schedule = empty()

  // cron format: minute(0-59) hour(0-23) day(1-31) month(1-12) weekday(0-6 0=sunday)
  const parts = cron.split(" ")
  if (parts.length !== 5) {
    throw new Error(`invalid cron expression: ${cron}`)
  }

  const [, hourField, , , weekdayField] = parts

  const hours = cronParseField(hourField)
  if (hours === null) {
    for (let i = 0; i < 24; i++) {
      schedule[`hour${i}`] = true
    }
  } else {
    for (const h of hours) {
      if (h >= 0 && h <= 23) {
        schedule[`hour${h}`] = true
      }
    }
  }

  const days = cronParseField(weekdayField)
  if (days === null) {
    for (let i = 0; i < 7; i++) {
      schedule[`day${i}`] = true
    }
  } else {
    for (const d of days) {
      if (d >= 0 && d <= 6) {
        schedule[`day${d}`] = true
      }
    }
  }

  return schedule
}

function toCron(schedule) {
  // cron format: minute(0-59) hour(0-23) day(1-31) month(1-12) weekday(0-6 0=sunday)

  let hours = []
  let days = []

  for (const key in schedule) {
    if (!schedule[key]) {
      continue
    }
    if (key.startsWith("day")) {
      const num = parseInt(key.slice(3), 10)
      days.push(num)
    } else if (key.startsWith("hour")) {
      const num = parseInt(key.slice(4), 10)
      hours.push(num)
    }
  }

  if (hours.length === 0) {
    return "invalid: at least one hour must be chosen"
  }

  if (days.length === 0) {
    return "invalid: at least one day must be chosen"
  }

  const min = "0" // always on the 0th minute
  const hour = cronMergeConsecutive(hours)
  const day = "*" // every day
  const month = "*" // every month
  const weekday = cronMergeConsecutive(days)

  return `${min} ${hour} ${day} ${month} ${weekday}`
}

window.schedule = {
  empty: empty,
  everyMorning: everyMorning,
  everyHour: everyHour,
  everyWeek: everyWeek,
  fromCron: fromCron,
  toCron: toCron,
}
