import * as MadNews from 'mad-news';

async function run() {
  // @ts-ignore
  const Madness = new MadNews('ru');
  console.log(Madness.fullString);
  Madness.generate();
  return Madness.fullString;
}

(async () => {
  const message = await run();
  console.log(message);
  process.exit(1);
})();
