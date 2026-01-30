import { Body, Controller, HttpStatus, Post, Get, Res } from '@nestjs/common';
import { WebhookService } from './webhook.service';
import { Response } from 'express';

import getCovid19 from './getCovid19';
import getNasaApod from './getNasaApod';

@Controller('api/webhooks')
export class WebhookController {
  @Post('/mad-news')
  async madNews(
    @Res() res: Response,
    @Body() body: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(body);
    res.status(HttpStatus.OK).json(result);
  }

  @Get('/covid19')
  async covid19(
    @Res() res: Response,
  ): Promise<any> {
    const result = await getCovid19();
    res.status(HttpStatus.OK).json(result);
  }

  @Get('/nasaapod')
  async nasaApod(
    @Res() res: Response,
  ): Promise<any> {
    const result = await getNasaApod();
    res.status(HttpStatus.OK).json(result);
  }

  @Post('/mad-news-wa')
  async madNewsWA(
    @Res() res: Response,
    @Body() body: any,
  ): Promise<any> {
    console.log('[WA] body:', body.Body);
    // if (['news', 'mad', 'madnews'].indexOf(req.Body) < 0) {
    //   return;
    // }
    const result = await WebhookService.sendWAReply();
    res.writeHead(200, { 'Content-Type': 'text/xml' });
    res.end(result.toString());
  }

  @Post('/mad-news-en')
  async madNewsEn(
    @Res() res: Response,
    @Body() body: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(body);
    res.status(HttpStatus.OK).json(result);
  }

  @Post('/vovan')
  async vovan(
    @Res() res: Response,
    @Body() body: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(body);
    res.status(HttpStatus.OK).json(result);
  }

  @Post('/hire')
  async hire(
    @Res() res: Response,
    @Body() body: any,
  ): Promise<any> {
    const result = await WebhookService.sendHireMe(body);
    res.status(HttpStatus.OK).json(result);
  }
}
