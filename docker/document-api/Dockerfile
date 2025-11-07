FROM tiangolo/uvicorn-gunicorn:python3.9

# create the appropriate directories
ENV HOME=/app

# create directory for the app user
RUN mkdir -p $HOME

# create the app user
RUN addgroup --system rapida-app && adduser --system --group rapida-app

# create app dir
ENV APP_HOME=$HOME

# as work dir
WORKDIR $APP_HOME/

RUN chown rapida-app:rapida-app $APP_HOME


ENV PYTHONDONTWRITEBYTECODE 1
ENV PYTHONUNBUFFERED 1

USER rapida-app

# copy non dev requirements
COPY --chown=rapida-app:rapida-app requirements.txt requirements.txt

# install dependencies
RUN pip install --user -r ./requirements.txt

# COPY PROJECT
COPY --chown=rapida-app:rapida-app . .

# give permission to execute scripts
# RUN chmod +x $APP_HOME/bin
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "9010"]
