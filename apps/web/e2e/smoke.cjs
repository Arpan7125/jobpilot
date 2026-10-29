/* eslint-disable @typescript-eslint/no-require-imports */
// Browser smoke test of the Week 1 flows. Needs the API and web app running,
// a database without arpan@example.com, and Playwright (npx playwright install chromium).
//   node e2e/smoke.cjs [screenshot-dir]
const { chromium } = require('playwright');
const BASE = process.env.BASE_URL ?? 'http://localhost:3000';
(async () => {
  const shots = process.argv[2] ?? '.';
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
  const log = (m) => console.log('✓', m);

  await page.goto(BASE + '/');
  await page.waitForURL('**/login'); log('logged-out visit redirects to /login');

  await page.click('text=Sign up');
  await page.fill('input[name=email]:visible', 'arpan@example.com');
  await page.fill('input[name=password]:visible', 'short');
  await page.fill('input[name=password]:visible', 'password123');
  await page.click('button:has-text("Create account") >> visible=true');
  await page.waitForURL(BASE + '/');
  await page.waitForSelector('text=No applications yet'); log('signup lands on empty board');

  const cookies = await page.context().cookies();
  const c = cookies.find(c => c.name === 'jp_token');
  if (!c || !c.httpOnly) throw new Error('token cookie missing or not httpOnly');
  log('JWT stored in httpOnly cookie');

  const add = async (company, title, extra = {}) => {
    await page.click('text=Add job');
    await page.fill('input[name=company]:visible', company);
    await page.fill('input[name=title]:visible', title);
    await page.fill('textarea[name=description]:visible', 'We need Go, PostgreSQL and Docker. 0-1 years.');
    for (const [k, v] of Object.entries(extra)) await page.fill(`input[name=${k}]:visible`, v);
    await page.click('button:has-text("Save job") >> visible=true');
    await page.waitForURL(BASE + '/');
  };

  // Validation error shown from the API
  await page.click('text=Add job');
  await page.fill('input[name=company]:visible', 'Bad Pay Co');
  await page.fill('input[name=title]:visible', 'Intern');
  await page.fill('textarea[name=description]:visible', 'x');
  await page.fill('input[name=salaryMin]:visible', '50000');
  await page.fill('input[name=salaryMax]:visible', '1000');
  await page.click('button:has-text("Save job") >> visible=true');
  await page.waitForSelector('text=must be at least salaryMin'); log('API validation error shown on form');
  if (await page.inputValue('input[name=company]:visible') !== 'Bad Pay Co') throw new Error('form lost typed values');
  log('typed values kept after validation error');

  await add('Acme', 'Backend Intern', { location: 'Bengaluru', platform: 'LinkedIn', salaryMin: '20000', salaryMax: '30000', currency: 'inr' });
  await add('Zeta', 'SDE Intern', { platform: 'Internshala' });
  await add('Razorpay', 'Platform Engineering Intern', { platform: 'Referral' });
  log('added 3 jobs');

  // Move Acme to applied from the board
  const card = page.locator('li', { hasText: 'Backend Intern' });
  await card.locator('select').selectOption('applied');
  await page.locator('section', { hasText: 'Applied' }).locator('text=Backend Intern').waitFor();
  log('board select moves card to Applied');

  const zeta = page.locator('li', { hasText: 'SDE Intern' });
  await zeta.locator('select').selectOption('interview');
  await page.locator('section', { hasText: 'Interview' }).locator('text=SDE Intern').waitFor();

  await page.screenshot({ path: `${shots}/board.png`, fullPage: true });

  // Detail page: status with note, history, edit
  await page.click('text=Backend Intern');
  await page.waitForSelector('h2:has-text("History")');
  await page.selectOption('select[name=status]:visible', 'replied');
  await page.fill('input[name=note]:visible', 'Recruiter emailed back');
  await page.click('button:has-text("Update") >> visible=true');
  await page.waitForSelector('text=Applied → Replied');
  await page.waitForSelector('text=Recruiter emailed back'); log('status change with note appears in history');
  if (await page.inputValue('select[name=status]:visible') !== 'replied') throw new Error('status select is stale');
  await page.screenshot({ path: `${shots}/job.png`, fullPage: true });

  await page.click('text=Edit');
  await page.fill('input[name=title]:visible', 'Backend Engineering Intern');
  await page.click('button:has-text("Save changes") >> visible=true');
  await page.waitForSelector('h1:has-text("Backend Engineering Intern")'); log('edit saves');

  // Delete
  await page.goto(BASE + '/');
  await page.click('text=Platform Engineering Intern');
  await page.click('button:has-text("Delete") >> visible=true');
  await page.waitForURL(BASE + '/');
  await page.locator('li a', { hasText: 'Platform Engineering Intern' }).waitFor({ state: 'detached', timeout: 5000 });
  log('delete removes job');

  await page.click('text=Log out');
  await page.waitForURL('**/login');
  await page.goto(BASE + '/');
  await page.waitForURL('**/login'); log('logout clears session');

  await page.fill('input[name=email]:visible', 'arpan@example.com');
  await page.fill('input[name=password]:visible', 'wrongpassword');
  await page.click('button:has-text("Log in") >> visible=true');
  await page.waitForSelector('text=wrong email or password'); log('wrong password rejected');
  await page.fill('input[name=password]:visible', 'password123');
  await page.click('button:has-text("Log in") >> visible=true');
  await page.waitForURL(BASE + '/'); await page.locator('li a', { hasText: 'Backend Engineering Intern' }).waitFor(); log('login shows saved jobs');

  await browser.close();
})().catch(e => { console.error('FAIL', e.message); process.exit(1); });
