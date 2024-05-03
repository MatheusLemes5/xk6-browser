import { check } from 'k6';
import { browser } from 'k6/x/browser';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      options: {
        browser: {
            type: 'chromium',
        },
      },
    },
  },
  thresholds: {
    checks: ["rate==1.0"]
  }
};

export default async function () {
  const page = browser.newPage();
  const context = page.context();

  try {
    // get cookies from the browser context
    check(context.cookies().length, {
        'initial number of cookies should be zero': n => n === 0,
    });

    // add some cookies to the browser context
    const unixTimeSinceEpoch = Math.round(new Date() / 1000);
    const day = 60*60*24;
    const dayAfter = unixTimeSinceEpoch+day;
    const dayBefore = unixTimeSinceEpoch-day;
    await context.addCookiesAsync([
      // this cookie expires at the end of the session
      {
        name: 'testcookie',
        value: '1',
        sameSite: 'Strict',
        domain: '127.0.0.1',
        path: '/',
        httpOnly: true,
        secure: true,
      },
      // this cookie expires in a day
      {
        name: 'testcookie2', 
        value: '2', 
        sameSite: 'Lax', 
        domain: '127.0.0.1', 
        path: '/', 
        expires: dayAfter,
      },
      // this cookie expires in the past, so it will be removed.
      {
        name: 'testcookie3',
        value: '3',
        sameSite: 'Lax',
        domain: '127.0.0.1',
        path: '/',
        expires: dayBefore
      }
    ]);
    let cookies = context.cookies();
    check(cookies.length, {
      'number of cookies should be 2': n => n === 2,
    });
    check(cookies[0], {
      'cookie 1 name should be testcookie': c => c.name === 'testcookie',
      'cookie 1 value should be 1': c => c.value === '1',
      'cookie 1 should be session cookie': c => c.expires === -1,
      'cookie 1 should have domain': c => c.domain === '127.0.0.1',
      'cookie 1 should have path': c => c.path === '/',
      'cookie 1 should have sameSite': c => c.sameSite == 'Strict',
      'cookie 1 should be httpOnly': c => c.httpOnly === true,
      'cookie 1 should be secure': c => c.secure === true,
    });
    check(cookies[1], {
      'cookie 2 name should be testcookie2': c => c.name === 'testcookie2',
      'cookie 2 value should be 2': c => c.value === '2',
    });

    // let's add more cookies to filter by urls.
    await context.addCookiesAsync([
      {
        name: "foo",
        value: "42",
        sameSite: "Strict",
        url: "http://foo.com",
      },
      {
        name: "bar",
        value: "43",
        sameSite: "Lax",
        url: "https://bar.com",
      },
      {
        name: "baz",
        value: "44",
        sameSite: "Lax",
        url: "https://baz.com",
      },
    ]);
    cookies = context.cookies('http://foo.com', 'https://baz.com');
    check(cookies.length, {
      'number of filtered cookies should be 2': n => n === 2,
    });
    check(cookies[0], {
      'the first filtered cookie name should be foo': c => c.name === 'foo',
      'the first filtered cookie value should be 42': c => c.value === '42',
    });
    check(cookies[1], {
      'the second filtered cookie name should be baz': c => c.name === 'baz',
      'the second filtered cookie value should be 44': c => c.value === '44',
    });

    // clear cookies
    context.clearCookies();
    cookies = context.cookies();
    check(cookies.length, {
      'number of cookies should be zero': n => n === 0,
    });
  } finally {
    page.close();
  }
}
