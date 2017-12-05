#!/usr/bin/python2.7
# -*- coding: utf-8 -*-

import os
import re
import logging
import random
import json
import uuid
import zipfile
import cStringIO
from collections import Counter

import webapp2
import yaml

from datetime import datetime

import cloudstorage as gcs

from google.appengine.api import background_thread
from google.appengine.api import app_identity
from google.appengine.api import taskqueue
from google.appengine.ext import deferred
from google.appengine.api import mail
from google.appengine.ext import ndb

from ete3 import Tree


MIN_SAMPLE_SIZE = 100
MAX_TREE_SIZE = 10000

SOURCE_PATH = '/data.vertlife.org/processing/%s/sources/single_trees/%s'
BASE_OUTPUT_PATH = '/data.vertlife.org/pruned_treesets'

JOB_CONFIG_PATH = '/data.vertlife.org/pruned_treesets/%s/config.yaml'
JOB_STATUS_PATH = '/data.vertlife.org/pruned_treesets/%s/status.log'
JOB_TREE_FILE_PATH = '/data.vertlife.org/pruned_treesets/%s/output.tre'
JOB_BAD_NAMES_PATH = '/data.vertlife.org/pruned_treesets/%s/bad_names.log'
JOB_ZIP_PATH = '/data.vertlife.org/pruned_treesets/{0}/{0}_output.zip'

TREE_CODES = {
    'EricsonStage2Full': 'Ericson All Species', 
    'EricsonStage1Full': 'Ericson Sequenced Species', 
    'HackettStage2Full': 'Hackett All Species',
    'HackettStage1Full': 'Hackett Sequenced Species', 
    'Stage2_DecisiveParrot': 'Stage2 Parrot', 
    'Stage2_FPTrees_EricsonDecisive': 'Stage2 FP Trees Ericson', 
    'Stage2_FPTrees_HackettDecisive': 'Stage2 FP Trees Hackett', 
    'Stage2_MayrAll_Ericson_decisive': 'Stage2 MayrAll Ericson', 
    'Stage2_MayrParSho_Ericson_decisive': 'Stage2 MayrParSho Ericson', 
    'Stage2_MayrAll_Hackett_decisive': 'Stage2 MayrAll Hackett', 
    'Stage2_MayrParSho_Hackett_decisive': 'Stage2 MayrParSho Hackett'
}
TREE_SITE_CODES = {
    'birdtree': 'BirdTree', 
    'sharktree': 'SharkTree'
}
TREE_SITE_URLS = {
    'birdtree': 'birdtree.org', 
    'sharktree': 'sharktree.org'
}


def write_file(filename, contents, content_type='text/plain'):
    write_retry_params = gcs.RetryParams(backoff_factor=1.1)
    with gcs.open(filename, 'w', 
        content_type=content_type, 
        retry_params=write_retry_params) as gcs_file:
        gcs_file.write(contents)


def start_prune_job(job_id):  # , email, tree_base, tree_set, sample_size, names
    with gcs.open(JOB_CONFIG_PATH % job_id) as fh:
        job_info = yaml.load(fh)    

    # init_job(job_id, email, tree_base, tree_set, sample_size, names)
    
    # Get the base tree from config
    tree_base = job_info['base_tree'].lower()

    # Get the tree_set from config
    tree_set = job_info['tree_set']
    for k, v in TREE_CODES.iteritems():
        if v == tree_set:
            tree_set = k
    
    names = [n.replace(' ', '_') for n in job_info['names']]
    sample_trees = job_info['sample_trees']
            
    start_pruning(job_id, tree_base, sample_trees, names)
    # process_bad_names(job_id)
    finalise_job(job_id, sample_trees, tree_base, tree_set)


def init_job(job_id, email, tree_base, tree_set, sample_size, names):
    tree_nums = random.sample(xrange(0, MAX_TREE_SIZE), sample_size)
    sample_trees = ["%s_%04d.tre" % (tree_set, n) for n in tree_nums]

    # Start by creating a config file
    job_info = {
        'job_id': job_id,
        'user_email': email,
        'base_tree': TREE_SITE_CODES[tree_base],
        'tree_set': TREE_CODES[tree_set],
        'sample_size': sample_size,
        'sample_trees': sample_trees,
        'names': [n.replace('_', ' ') for n in names],
        'status': 'CREATED',
        'created_at': datetime.utcnow().strftime('%Y-%m-%d %H:%M')
    }
    write_file(JOB_CONFIG_PATH % job_id, yaml.dump(job_info))
    
    # Initialize the status file
    write_file(JOB_STATUS_PATH % job_id, 'STARTED')


def start_pruning(job_id, tree_base, sample_trees, names):
    try:

        logging.info('Processing names: ')
        logging.info(names)
        bad_names = []
        for idx, tree_file in enumerate(sample_trees):
            logging.info("Pruning tree: %s, %s" % (idx, tree_file))
            out_filename = '%s/%s/pruned/%s' % (BASE_OUTPUT_PATH, job_id, tree_file)

            try:
                with gcs.open(SOURCE_PATH % (tree_base, tree_file)) as gcs_file:
                    tree_file = gcs_file.read()

                tree = Tree(tree_file)
                try:
                    tree.prune(names, preserve_branch_length=True)
                except Exception as a:
                    missing_names = str(a).replace('Node names not found: ','')
                    valid_names = filter(lambda name: name not in missing_names, names)
                    tree.prune(valid_names, preserve_branch_length=True)

                    # add the bad names to a list
                    mn = ''.join(n for n in missing_names if (n.isalpha() or n==',' or n=='_')).split(',')
                    bad_names.extend(mn)

                write_file(out_filename, tree.write())
            except Exception as e:
                logging.error(e)
                write_file(out_filename, 'NO DATA')

        # Write out the bad names
        logging.debug('all_bad_names: %s' % bad_names)
        really_bad_names = [x for x, y in Counter(bad_names).items() if y == len(sample_trees)]
        logging.debug('really_bad_names: %s' % really_bad_names)
        write_file(JOB_BAD_NAMES_PATH % job_id, yaml.dump({
            'bad_names': [n.replace('_', ' ') for n in bad_names], 
            'really_bad_names': [n.replace('_', ' ') for n in really_bad_names]
        }))

        # Finished with pruning. Let's set the status
        write_file(JOB_STATUS_PATH % job_id, 'PRUNED')

    except Exception as e:
        logging.error(e)
        err_filename = '/%s/%s/error.log' % (BASE_OUTPUT_PATH, job_id)
        write_file(err_filename, e)
        write_file(JOB_STATUS_PATH % job_id, 'ERROR')


def _get_header(site_url, created_at, tree_set):
    return """#NEXUS

[Tree distribution from: The global diversity of birds in space and time; W. Jetz, G. H. Thomas, J. B. Joy, K. Hartmann, A. O. Mooers doi:10.1038/nature11631]
[Subsampled and pruned from {} on {} ]
[Data: "{}" (see Jetz et al. 2012 supplement for details)]

BEGIN TREES;
    """.format(site_url, created_at, tree_set)


def finalise_job(job_id, sample_trees, tree_base, tree_set):
    # read the config file 
    with gcs.open(JOB_CONFIG_PATH % job_id) as fh:
        job_info = yaml.load(fh)

    with gcs.open(JOB_BAD_NAMES_PATH % job_id) as fh:
        bad_names = yaml.load(fh)

    # stats = gcs.listbucket('%s/%s/' % (BASE_OUTPUT_PATH, job_id), max_keys=MAX_TREE_SIZE, delimiter='/')
    # with gcs.open(JOB_TREE_FILE_PATH % job_id) as cfh:
    #     cfh.write(_get_header(job_info['created_at'], job_info['tree_set']))
    #     for stat in stats:
    #         prune_tree_num = stat.filename.split('_')[-1].replace('.tre', '')
    #         with gcs.open(stat.filename) as fh:
    #             cfh.write('\n\tTREE tree_%s = ' % prune_tree_num)
    #             cfh.write(fh.read())
    #     cfh.write('\nEND;\n\n')

    output = cStringIO.StringIO()
    content = ''
    try:
        output.write(_get_header(TREE_SITE_URLS[tree_base], job_info['created_at'], TREE_CODES[tree_set]))
        for tree_file in sample_trees:
            pruned_tree_file = '%s/%s/pruned/%s' % (BASE_OUTPUT_PATH, job_id, tree_file)
            prune_tree_num = pruned_tree_file.split('_')[-1].replace('.tre', '')
            try:
                with gcs.open(pruned_tree_file) as fh:
                    output.write('\n\tTREE tree_%s = ' % prune_tree_num)
                    output.write(fh.read())
            except Exception as e:
                logging.error('Missing tree. Fix this: %s' % pruned_tree_file)
                output.write('\n\tTREE tree_%s = (NO TREE FILE. FIX THIS.);' % prune_tree_num)
        output.write('\nEND;\n\n')
    except Exception as e:
        logging.error('Unable to write output tre file')
        logging.error(e)
    finally:
        content = output.getvalue()
        output.close()
    write_file(JOB_TREE_FILE_PATH % job_id, content)

    # Update the config file
    job_info.update({
        'status': 'COMPLETED',
        'completed_at': datetime.utcnow().strftime('%Y-%m-%d %H:%M'),
        'bad_names': bad_names['really_bad_names']
    })
    write_file(JOB_CONFIG_PATH % job_id, yaml.dump(job_info))
    
    # Finalise the status file
    write_file(JOB_STATUS_PATH % job_id, 'COMPLETED')

    # Zip up the necessary files
    dl_file_path = JOB_ZIP_PATH.format(job_id)
    with gcs.open(dl_file_path, 'w', 
        content_type='application/zip', 
        options={'x-goog-acl': 'public-read'}) as czfh:
        with zipfile.ZipFile(czfh, 'w') as zfh:
            zfh.writestr('output.tre', content)
            zfh.writestr('config.yaml', yaml.dump(job_info))

    dl_url = 'http://storage.googleapis.com{}'.format(dl_file_path)

    # Notify the user
    sender_address = ('Map of Life <{}@appspot.gserviceaccount.com>'.format(app_identity.get_application_id()))
    subject = '{}: Your pruned trees are ready'.format(TREE_SITE_CODES[tree_base])
    body = """Thank you for using the {0} service to prune your trees.
You can access your pruned trees and additional information here:
  Pruned Trees: {1}
  TasK ID: {2}
    """.format(TREE_SITE_CODES[tree_base], dl_url, job_id)
    mail.send_mail(sender_address, job_info['user_email'], subject, body)


def handle_400(request, response, exception):
    # logging.exception(exception)
    response.set_status(400)

def handle_404(request, response, exception):
    logging.exception(exception)
    response.write('Oops! I could swear this page was here!')
    response.set_status(404)

def handle_500(request, response, exception):
    logging.exception(exception)
    response.write('A server error occurred!')
    response.set_status(500)

class PruningJobHandler(webapp2.RequestHandler):
    def get(self):
        self.process_request()

    def post(self):
        self.process_request()

    def send_error(self, message):
        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'application/json'
        self.response.out.write(json.dumps({'status': 'error', 'message': message}))
        webapp2.abort(400)

    def process_request(self):

        # Grab the necessary params
        sample_size = int(self.request.get('sample_size', MIN_SAMPLE_SIZE))
        tree_base = str(self.request.get('tree_base', 'birdtree')).replace('%20', ' ').strip()
        tree_set = str(self.request.get('tree_set', 'Stage2_Parrot')).replace('%20', ' ').strip()
        _species = self.request.get('species', None)
        email = self.request.get('email', None)

        if email is None or not (re.match(r"[^@]+@[^@]+\.[^@]+", email)):  # not mail.is_email_valid(email):
            self.send_error('Please provide a valid email address')

        if _species is None or len(_species.strip()) == 0:  # or names is None or len(names) == 0
            self.send_error('Please provide a valid set of species to prune')

        if sample_size < MIN_SAMPLE_SIZE or sample_size > MAX_TREE_SIZE:
            self.send_error('Please select a sample size between %d and %d' % (MIN_SAMPLE_SIZE, MAX_TREE_SIZE))

        job_id = 'tree-pruner-%s' % uuid.uuid4()
        email = str(email).replace('%20', ' ').strip()
        names = set(map(
            lambda n: n.replace('%20', ' ').strip().replace(' ', '_'),
            str(_species).replace('\n', ',').replace(', ',',').split(',')
        ))
        # background_thread.start_new_background_thread(start_prune_job, [job_id, email, tree_base, tree_set, sample_size, names])
        init_job(job_id, email, tree_base, tree_set, sample_size, names)
        deferred.defer(start_prune_job, job_id, _queue='pruner')

        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'application/json'
        self.response.out.write(json.dumps({'status': 'created', 'job_id': job_id}))


class PruningResultHandler(webapp2.RequestHandler):
    def get(self):
        job_id = self.request.get('job_id')
        email = self.request.get('email')

        if email is None or not (re.match(r"[^@]+@[^@]+\.[^@]+", email)): 
            self.response.headers.add_header("Access-Control-Allow-Origin", "*")
            self.response.headers['Content-Type'] = 'application/json'
            self.response.out.write(json.dumps({'status': 'error', 'message': 'Please provide a valid email address'}))
            webapp2.abort(400)

        if job_id is None or len(job_id.strip()) == 0:  
            self.response.headers.add_header("Access-Control-Allow-Origin", "*")
            self.response.headers['Content-Type'] = 'application/json'
            self.response.out.write(json.dumps({'status': 'error', 'message': 'Please provide a valid job id'}))
            webapp2.abort(400)

        # try:
        #     with gcs.open(JOB_TREE_FILE_PATH % job_id) as fh:
        #         self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        #         self.response.headers['Content-Type'] = 'text/plain'
        #         self.response.out.write(fh.read())
        # except gcs.NotFoundError as nfe:
        #     # TODO: Read the status and provide the appropriate response
        #     self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        #     self.response.headers['Content-Type'] = 'application/json'
        #     self.response.out.write(json.dumps({'status': 'pruning', 'job_id': job_id}))

        # TODO: Check that both the job_id and email match
        message = {}
        email = str(email).replace('%20', ' ').strip()
        job_id = str(job_id).replace('%20', ' ').strip()
        try:
            with gcs.open(JOB_STATUS_PATH % job_id) as fh:
                status = fh.read()
            if status == 'COMPLETED':
                dl_file_path = JOB_ZIP_PATH.format(job_id)
                message = {
                    'status': 'completed', 
                    'message': 'http://storage.googleapis.com{}'.format(dl_file_path)
                }
            elif status == 'error':

                # Check if there is an error file
                error_message = 'There was a problem generating samples. Please try again or contact us if the problem persists'
                try:
                    err_filename = '/{}/{}/error.log'.format(BASE_OUTPUT_PATH, job_id)
                    with gcs.open(err_filename) as fh:
                        error_message = fh.read()
                except Exception as e:
                    pass

                message = {
                    'status': 'error', 
                    'message': error_message
                }
            else:
                message = {
                    'status': 'purning', 
                    'message': 'Please wait while we generate some samples'
                }

        except gcs.NotFoundError as nfe:
            message = {'status': 'error', 'message': 'Are you sure you have a valid Job ID and email? Please try again or contact us if the problem persists'}

        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'application/json'
        self.response.out.write(json.dumps(message))



application = webapp2.WSGIApplication(
    [('/api/prune', PruningJobHandler),
     ('/api/result', PruningResultHandler)
    ], debug=True)
application.error_handlers[400] = handle_400
application.error_handlers[404] = handle_404
application.error_handlers[500] = handle_500
