#!/usr/bin/python2.7
# -*- coding: utf-8 -*-

import os
import logging
import random
import json
import uuid
import webapp2

import cloudstorage as gcs

from google.appengine.api import app_identity
from google.appengine.api import taskqueue
from google.appengine.ext import deferred
from google.appengine.ext import ndb

from ete2 import Tree


MIN_SAMPLE_SIZE = 100
MAX_TREE_SIZE = 10000
SOURCE_PATH = '/data.vertlife.org/processing/%s/sources/single_trees/%s'

def create_file(filename, contents):
    """Create a file.

    The retry_params specified in the open call will override the default
    retry params for this particular file handle.

    Args:
    filename: filename.
    """
    write_retry_params = gcs.RetryParams(backoff_factor=1.1)
    gcs_file = gcs.open(filename,
        'w',
        content_type='text/plain',
        options={'x-goog-meta-foo': 'foo',
                'x-goog-meta-bar': 'bar'},
        retry_params=write_retry_params)
    gcs_file.write(contents)
    gcs_file.close()


def processTree(tree_base, tree_filename, job_id, names):
    try:
        gcs_file = gcs.open(SOURCE_PATH % (tree_base, tree_filename))
        tree_file = gcs_file.read()
        gcs_file.close()
        tree = Tree(tree_file)
        logging.info(names)
        try:
            tree.prune(names, preserve_branch_length=True)
        except Exception as a:
            missing_names = str(a).replace('Node names not found: ','')
            names = filter(lambda name: name not in missing_names, names)
            tree.prune(names)

        filename = '/data.vertlife.org/pruned_treesets/%s/%s' % (job_id, tree_filename)
        create_file(filename, tree.write())
    except Exception as e:
        logging.error(e)
        create_file(filename, 'NO DATA')


class PruningJobHandler(webapp2.RequestHandler):
    def get(self):
        self.process_request()

    def post(self):
        self.process_request()

    def process_request(self):

        def start_tasks(tree_base, sample_trees):
            for idx, tree_file in enumerate(sample_trees):
                logging.info("Spawned task %s, %s" % (idx, tree_file))
                deferred.defer(processTree, tree_base, tree_file, job_id, names, 
                    _queue='pruner')

        # Grab the necessary params
        sample_size = int(self.request.get('sample_size', MIN_SAMPLE_SIZE))
        tree_base = str(self.request.get('tree_base', 'birdtree'))
        tree_set = str(self.request.get('tree_set', 'Stage2_Parrot'))
        names = map(
            lambda n: n.replace('%20', ' ').strip().replace(' ', '_'),
            str(self.request.get('species', '')).replace(', ',',').split(',')
        )

        job_id = 'tree-pruner-%s' % uuid.uuid4()
        tree_nums = random.sample(xrange(0, MAX_TREE_SIZE), sample_size)
        
        sample_trees = ["%s_%04d.tre" % (tree_set, n) for n in tree_nums]
        start_tasks(tree_base, sample_trees)

        self.response.headers.add_header("Access-Control-Allow-Origin", "*")
        self.response.headers['Content-Type'] = 'application/json'
        self.response.out.write(json.dumps({'job_id': job_id}))


class PruningResultHandler(webapp2.RequestHandler):
    def get(self):
        folder = '/data.vertlife.org/pruned_treesets'
        job_id = self.request.get('job_id')
        trees = gcs.listbucket(folder + '/' + job_id, max_keys=10000)
        for tree in trees:
            pruned_tree = gcs.open(tree)
            self.response.out.write(pruned_tree.read())
            pruned_tree.close()


application = webapp2.WSGIApplication(
    [('/api/prune', PruningJobHandler),
     ('/api/result', PruningResultHandler)
    ], debug=True)
