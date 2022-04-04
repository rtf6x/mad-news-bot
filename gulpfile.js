const gulp = require('gulp');
const backWorker = require('./workers/background_workers/index');

const newQuestions = done => {
  backWorker.newQuestions(done);
};
const finish = done => {
  done();
  process.exit(1);
};

module.exports.newQuestions = gulp.series(newQuestions, finish);
module.exports.default = finish;
