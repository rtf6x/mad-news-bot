import { Body, Controller, HttpStatus, Post, Get, Res } from '@nestjs/common';
import { WebhookService } from './webhook.service';
import { Response } from 'express';

@Controller('api/webhooks')
export class WebhookController {
  @Post('/mad-news')
  async madNews(
    @Res() res: Response,
    @Body() req: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(req);
    res.status(HttpStatus.OK).json(result);
  }

  @Get('/covid19')
  async covid19(
    @Res() res: Response,
  ): Promise<any> {
    const result = await WebhookService.getCovid2();
    res.status(HttpStatus.OK).json(result);
  }

  @Post('/mad-news-wa')
  async madNewsWA(
    @Res() res: Response,
    @Body() req: any,
  ): Promise<any> {
    console.log('[WA] body:', req.Body);
    // if (['news', 'mad', 'madnews'].indexOf(req.Body) < 0) {
    //   return;
    // }
    const result = await WebhookService.sendWAReply();
    res.writeHead(200, { 'Content-Type': 'text/xml' });
    res.end(result.toString());
  }

  @Post('mad-news-en')
  async madNewsEn(
    @Res() res: Response,
    @Body() req: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(req);
    res.status(HttpStatus.OK).json(result);
  }

  @Post('vovan')
  async vovan(
    @Res() res: Response,
    @Body() req: any,
  ): Promise<any> {
    const result = await WebhookService.sendReply(req);
    res.status(HttpStatus.OK).json(result);
  }
}
